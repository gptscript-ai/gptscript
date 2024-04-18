package runner

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type dispatcher interface {
	Run(func(context.Context) error)
	Wait() error
}

type serialDispatcher struct {
	ctx context.Context
	err error
}

func newSerialDispatcher(ctx context.Context) *serialDispatcher {
	return &serialDispatcher{
		ctx: ctx,
	}
}

func (s *serialDispatcher) Run(f func(context.Context) error) {
	if s.err != nil {
		return
	}
	s.err = f(s.ctx)
}

func (s *serialDispatcher) Wait() error {
	return s.err
}

type parallelDispatcher struct {
	ctx context.Context
	eg  *errgroup.Group
}

func newParallelDispatcher(ctx context.Context) *parallelDispatcher {
	eg, ctx := errgroup.WithContext(ctx)
	return &parallelDispatcher{
		ctx: ctx,
		eg:  eg,
	}
}

func (p *parallelDispatcher) Run(f func(context.Context) error) {
	p.eg.Go(func() error {
		return f(p.ctx)
	})
}

func (p *parallelDispatcher) Wait() error {
	return p.eg.Wait()
}
