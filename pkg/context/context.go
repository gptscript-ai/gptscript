package context

import (
	"context"

	"github.com/google/uuid"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
)

type pauseKey struct{}

func AddPauseFuncToCtx(ctx context.Context, pauseF func() func()) context.Context {
	return context.WithValue(ctx, pauseKey{}, pauseF)
}

func GetPauseFuncFromCtx(ctx context.Context) func() func() {
	pauseF, ok := ctx.Value(pauseKey{}).(func() func())
	if !ok {
		return func() func() { return func() {} }
	}
	return pauseF
}

type reqIDKey struct{}

func WithNewRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, reqIDKey{}, uuid.NewString())
}

func GetRequestID(ctx context.Context) string {
	s, _ := ctx.Value(reqIDKey{}).(string)
	return s
}

type loggerKey struct{}

func WithLogger(ctx context.Context, log mvl.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, log)
}

func GetLogger(ctx context.Context) mvl.Logger {
	l, ok := ctx.Value(loggerKey{}).(mvl.Logger)
	if !ok {
		return mvl.New("")
	}

	return l
}
