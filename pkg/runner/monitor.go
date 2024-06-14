package runner

import (
	"context"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type noopFactory struct {
}

func (n noopFactory) Start(context.Context, *types.Program, []string, string) (Monitor, error) {
	return noopMonitor{}, nil
}

func (n noopFactory) Pause() func() {
	return func() {}
}

type noopMonitor struct {
}

func (n noopMonitor) Event(Event) {
}

func (n noopMonitor) Stop(string, error) {}

func (n noopMonitor) Pause() func() {
	return func() {}
}

type credWrapper struct {
	m Monitor
}

func (c credWrapper) Event(e Event) {
	if e.Type == EventTypeCallFinish {
		e.Content = "credential tool output redacted"
	}
	c.m.Event(e)
}

func (c credWrapper) Stop(s string, err error) {
	c.m.Stop(s, err)
}

func (c credWrapper) Pause() func() {
	return c.m.Pause()
}
