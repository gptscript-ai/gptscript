package sdkserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/auth"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	gserver "github.com/gptscript-ai/gptscript/pkg/server"
)

func (s *server) authorize(ctx engine.Context, input string) (runner.AuthorizerResponse, error) {
	defer gcontext.GetPauseFuncFromCtx(ctx.Ctx)()()

	if auth.IsSafe(ctx) {
		return runner.AuthorizerResponse{
			Accept: true,
		}, nil
	}

	s.lock.RLock()
	authChan := s.waitingToConfirm[ctx.ID]
	s.lock.RUnlock()

	if authChan != nil {
		return runner.AuthorizerResponse{}, fmt.Errorf("authorize called multiple times for same ID: %s", ctx.ID)
	}

	runID := gserver.RunIDFromContext(ctx.Ctx)
	s.lock.Lock()
	authChan = make(chan runner.AuthorizerResponse)
	s.waitingToConfirm[ctx.ID] = authChan
	s.lock.Unlock()
	defer func(id string) {
		s.lock.Lock()
		delete(s.waitingToConfirm, id)
		s.lock.Unlock()
	}(ctx.ID)

	s.events.C <- event{
		Event: gserver.Event{
			Event: runner.Event{
				Time:        time.Now(),
				CallContext: ctx.GetCallContext(),
				Type:        CallConfirm,
			},
			Input: input,
			RunID: runID,
		},
	}

	// Wait for the confirmation to come through.
	select {
	case <-ctx.Ctx.Done():
		return runner.AuthorizerResponse{}, ctx.Ctx.Err()
	case authResponse := <-authChan:
		return authResponse, nil
	}
}

func (s *server) confirm(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	id := r.PathValue("id")

	s.lock.RLock()
	authChan := s.waitingToConfirm[id]
	s.lock.RUnlock()

	if authChan == nil {
		writeError(logger, w, http.StatusNotFound, fmt.Errorf("no confirmation found with id %q", id))
		return
	}

	var authResponse runner.AuthorizerResponse
	if err := json.NewDecoder(r.Body).Decode(&authResponse); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	// Don't block here because, if the authorizer is no longer waiting on this then it will never unblock.
	select {
	case authChan <- authResponse:
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusConflict)
	}
}
