package server

import (
	"context"

	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Event struct {
	runner.Event `json:",inline"`
	RunID        string         `json:"runID,omitempty"`
	Program      *types.Program `json:"program,omitempty"`
	Input        string         `json:"input,omitempty"`
	Output       string         `json:"output,omitempty"`
	Err          string         `json:"err,omitempty"`
}

type execKey struct{}

func ContextWithNewRunID(ctx context.Context) context.Context {
	return context.WithValue(ctx, execKey{}, counter.Next())
}

func RunIDFromContext(ctx context.Context) string {
	return ctx.Value(execKey{}).(string)
}
