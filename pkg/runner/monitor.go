package runner

import (
	"context"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type noopFactory struct {
}

func (n noopFactory) Start(ctx context.Context, prg *types.Program, env []string, input string) (Monitor, error) {
	return noopMonitor{}, nil
}

type noopMonitor struct {
}

func (n noopMonitor) Event(event Event) {
}

func (n noopMonitor) Stop(output string, err error) {
}
