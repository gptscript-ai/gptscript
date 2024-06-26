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

func (s *server) execAndStream(ctx context.Context, programLoader loaderFunc, logger mvl.Logger, w http.ResponseWriter, opts gptscript.Options, chatState, input, subTool string, toolDef fmt.Stringer) {
	g, err := gptscript.New(ctx, s.gptscriptOpts, opts)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	prg, err := programLoader(ctx, toolDef.String(), subTool, loader.Options{Cache: g.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	errChan := make(chan error)
	programOutput := make(chan runner.ChatResponse)
	events := s.events.Subscribe()
	defer events.Close()

	go func() {
		run, err := g.Chat(ctx, chatState, prg, opts.Env, input)
		if err != nil {
			errChan <- err
		} else {
			programOutput <- run
		}
		close(errChan)
		close(programOutput)
	}()

	processEventStreamOutput(ctx, logger, w, gserver.RunIDFromContext(ctx), events.C, programOutput, errChan)
}

// processEventStreamOutput will stream the events of the tool to the response as server sent events.
// If an error occurs, then an event with the error will also be sent.
func processEventStreamOutput(ctx context.Context, logger mvl.Logger, w http.ResponseWriter, id string, events <-chan event, output <-chan runner.ChatResponse, errChan chan error) {
	run := newRun(id)
	setStreamingHeaders(w)

	streamEvents(ctx, logger, w, run, events)

	var out runner.ChatResponse
	select {
	case <-ctx.Done():
	case out = <-output:
		run.processStdout(out)

		writeServerSentEvent(logger, w, map[string]any{
			"stdout": out,
		})
	case err := <-errChan:
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run file: %w", err))
	}

	// Now that we have received all events, send the DONE event.
	_, err := w.Write([]byte("data: [DONE]\n\n"))
	if err == nil {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	logger.Debugf("wrote DONE event")
}

// streamEvents will stream the events of the tool to the response as server sent events.
func streamEvents(ctx context.Context, logger mvl.Logger, w http.ResponseWriter, run *runInfo, events <-chan event) {
	logger.Debugf("receiving events")
	for {
		select {
		case <-ctx.Done():
			logger.Debugf("context canceled while receiving events")
			go func() {
				//nolint:revive
				for range events {
				}
			}()
			return
		case e, ok := <-events:
			if ok && e.RunID != run.ID {
				continue
			}

			if !ok {
				logger.Debugf("done receiving events")
				return
			}

			writeServerSentEvent(logger, w, run.process(e))

			if e.Type == runner.EventTypeRunFinish {
				logger.Debugf("finished receiving events")
				return
			}
		}
	}
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
