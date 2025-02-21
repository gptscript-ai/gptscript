package sdkserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	gserver "github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type loaderFunc func(context.Context, string, string, ...loader.Options) (types.Program, error)

func loaderWithLocation(f loaderFunc, loc string) loaderFunc {
	return func(ctx context.Context, s string, s2 string, options ...loader.Options) (types.Program, error) {
		return f(ctx, s, s2, append(options, loader.Options{
			Location: loc,
		})...)
	}
}

func (s *server) execAndStream(ctx context.Context, programLoader loaderFunc, logger mvl.Logger, w http.ResponseWriter, opts gptscript.Options, chatState, input, subTool string, toolDef fmt.Stringer, cancel <-chan struct{}) {
	g, err := gptscript.New(ctx, s.gptscriptOpts, opts)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	defaultModel := opts.OpenAI.DefaultModel
	if defaultModel == "" {
		defaultModel = s.gptscriptOpts.OpenAI.DefaultModel
	}
	prg, err := programLoader(ctx, toolDef.String(), subTool, loader.Options{Cache: g.Cache, DefaultModel: defaultModel})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	errChan := make(chan error)
	programOutput := make(chan runner.ChatResponse)
	events := s.events.Subscribe()
	defer events.Close()

	go func() {
		run, err := g.Chat(ctx, chatState, prg, opts.Env, input, runner.RunOptions{
			UserCancel: cancel,
		})
		if err != nil {
			errChan <- err
		} else {
			programOutput <- run
		}
		close(errChan)
		close(programOutput)
	}()

	processEventStreamOutput(logger, w, gserver.RunIDFromContext(ctx), events.C, programOutput, errChan)
}

// processEventStreamOutput will stream the events of the tool to the response as server sent events.
// If an error occurs, then an event with the error will also be sent.
func processEventStreamOutput(logger mvl.Logger, w http.ResponseWriter, id string, events <-chan event, output <-chan runner.ChatResponse, errChan chan error) {
	run := newRun(id)
	setStreamingHeaders(w)

	streamEvents(logger, w, run, events)

	select {
	case out := <-output:
		run.processStdout(out)

		writeServerSentEvent(logger, w, map[string]any{
			"stdout": out,
		})
	case err := <-errChan:
		writeServerSentEvent(logger, w, map[string]any{
			"stderr": fmt.Sprintf("failed to run: %v", err),
		})
	}

	// Now that we have received all events, send the DONE event.
	writeServerSentEvent(logger, w, "[DONE]")

	logger.Debugf("wrote DONE event")
}

// streamEvents will stream the events of the tool to the response as server sent events.
func streamEvents(logger mvl.Logger, w http.ResponseWriter, run *runInfo, events <-chan event) {
	logger.Debugf("receiving events")
	for e := range events {
		if e.RunID != run.ID {
			continue
		}

		writeServerSentEvent(logger, w, run.process(e))

		if e.Type == runner.EventTypeRunFinish {
			break
		}
	}

	logger.Debugf("done receiving events")
}

func writeResponse(logger mvl.Logger, w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to marshal response: %w", err))
		return
	}

	_, _ = w.Write(b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func writeError(logger mvl.Logger, w http.ResponseWriter, code int, err error) {
	logger.Debugf("Writing error response with code %d: %v", code, err)

	w.WriteHeader(code)
	resp := map[string]any{
		"stderr": err.Error(),
	}

	b, err := json.Marshal(resp)
	if err != nil {
		_, _ = w.Write([]byte(fmt.Sprintf(`{"stderr": "%s"}`, err.Error())))
		return
	}

	_, _ = w.Write(b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func writeServerSentEvent(logger mvl.Logger, w http.ResponseWriter, event any) {
	ev, err := json.Marshal(event)
	if err != nil {
		logger.Warnf("failed to marshal event: %v", err)
		return
	}

	_, err = w.Write([]byte(fmt.Sprintf("data: %s\n\n", ev)))
	if err == nil {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	logger.Debugf("wrote event: %v", string(ev))
}

func setStreamingHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}
