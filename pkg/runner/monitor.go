package runner

import (
	"context"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type noopFactory struct{}

func (n noopFactory) Start(context.Context, *types.Program, []string, string) (Monitor, error) {
	return noopMonitor{}, nil
}

func (n noopFactory) Pause() func() {
	return func() {}
}

type noopMonitor struct{}

func (n noopMonitor) Event(Event) {
}

func (n noopMonitor) Stop(context.Context, string, error) {}

func (n noopMonitor) Pause() func() {
	return func() {}
}
