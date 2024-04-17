package context

import (
	"context"
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
