package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func runtimeWithLogger(callCtx engine.Context, monitor Monitor, rm engine.RuntimeManager) engine.RuntimeManager {
	if rm == nil {
		return nil
	}
	return runtimeManagerLogger{
		callCtx: callCtx,
		monitor: monitor,
		rm:      rm,
	}
}

type runtimeManagerLogger struct {
	callCtx engine.Context
	monitor Monitor
	rm      engine.RuntimeManager
}

func (r runtimeManagerLogger) Infof(msg string, args ...any) {
	r.monitor.Event(Event{
		Time:        time.Now(),
		Type:        EventTypeCallProgress,
		CallContext: r.callCtx.GetCallContext(),
		Content:     fmt.Sprintf(msg, args...),
	})
}

func (r runtimeManagerLogger) GetContext(ctx context.Context, tool types.Tool, cmd, env []string) (string, []string, error) {
	return r.rm.GetContext(mvl.WithInfo(ctx, r), tool, cmd, env)
}

func (r runtimeManagerLogger) EnsureCredentialHelpers(ctx context.Context) error {
	return r.rm.EnsureCredentialHelpers(mvl.WithInfo(ctx, r))
}

func (r runtimeManagerLogger) SetUpCredentialHelpers(_ context.Context, _ *config.CLIConfig, _ []string) error {
	panic("not implemented")
}
