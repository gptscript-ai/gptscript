package sdkserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	gserver "github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (s *server) promptResponse(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	id := r.PathValue("id")

	s.lock.RLock()
	promptChan := s.waitingToPrompt[id]
	s.lock.RUnlock()

	if promptChan == nil {
		writeError(logger, w, http.StatusNotFound, fmt.Errorf("no prompt found with id %q", id))
		return
	}

	var promptResponse map[string]string
	if err := json.NewDecoder(r.Body).Decode(&promptResponse); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	// Don't block here because, if the prompter is no longer waiting on this then it will never unblock.
	select {
	case promptChan <- promptResponse:
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusConflict)
	}
}

func (s *server) prompt(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	if r.Header.Get("Authorization") != "Bearer "+s.token {
		writeError(logger, w, http.StatusUnauthorized, fmt.Errorf("invalid token"))
		return
	}

	id := r.PathValue("id")

	s.lock.RLock()
	promptChan := s.waitingToPrompt[id]
	s.lock.RUnlock()

	if promptChan != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("prompt called multiple times for same ID: %s", id))
		return
	}

	var prompt types.Prompt
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %v", err))
		return
	}

	s.lock.Lock()
	promptChan = make(chan map[string]string)
	s.waitingToPrompt[id] = promptChan
	s.lock.Unlock()
	defer func(id string) {
		s.lock.Lock()
		delete(s.waitingToPrompt, id)
		s.lock.Unlock()
	}(id)

	s.events.C <- event{
		Prompt: types.Prompt{
			Message:   prompt.Message,
			Fields:    prompt.Fields,
			Sensitive: prompt.Sensitive,
		},
		Event: gserver.Event{
			RunID: id,
			Event: runner.Event{
				Type: Prompt,
				Time: time.Now(),
			},
		},
	}

	// Wait for the prompt response to come through.
	select {
	case <-r.Context().Done():
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("context canceled: %v", r.Context().Err()))
		return
	case promptResponse := <-promptChan:
		writePromptResponse(logger, w, http.StatusOK, promptResponse)
	}
}

func writePromptResponse(logger mvl.Logger, w http.ResponseWriter, code int, resp any) {
	b, err := json.Marshal(resp)
	if err != nil {
		logger.Errorf("failed to marshal response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(code)
	}

	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("failed to write response: %v", err)
	}
}
