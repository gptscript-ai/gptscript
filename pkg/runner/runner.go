package runner

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/gptscript/pkg/cache"
	"github.com/acorn-io/gptscript/pkg/engine"
	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/types"
	"golang.org/x/sync/errgroup"
)

type MonitorFactory interface {
	Start(ctx context.Context, prg *types.Program, env []string, input string) (Monitor, error)
}

type Monitor interface {
	Event(event Event)
	Stop()
}

type (
	CacheOptions  cache.Options
	OpenAIOptions openai.Options
)

type Options struct {
	CacheOptions
	OpenAIOptions
	MonitorFactory MonitorFactory `usage:"-"`
}

func complete(opts ...Options) (cacheOpts []cache.Options, oaOpts []openai.Options, result Options) {
	for _, opt := range opts {
		cacheOpts = append(cacheOpts, cache.Options(opt.CacheOptions))
		oaOpts = append(oaOpts, openai.Options(opt.OpenAIOptions))
		result.MonitorFactory = types.FirstSet(opt.MonitorFactory, result.MonitorFactory)
	}
	if result.MonitorFactory == nil {
		result.MonitorFactory = noopFactory{}
	}
	return
}

type Runner struct {
	c       *openai.Client
	factory MonitorFactory
}

func New(opts ...Options) (*Runner, error) {
	cacheOpts, oaOpts, opt := complete(opts...)
	cacheClient, err := cache.New(cacheOpts...)
	if err != nil {
		return nil, err
	}

	oaClient, err := openai.NewClient(append(oaOpts, openai.Options{
		Cache: cacheClient,
	})...)
	if err != nil {
		return nil, err
	}

	return &Runner{
		c:       oaClient,
		factory: opt.MonitorFactory,
	}, nil
}

func (r *Runner) Run(ctx context.Context, prg types.Program, env []string, input string) (string, error) {
	monitor, err := r.factory.Start(ctx, &prg, env, input)
	if err != nil {
		return "", err
	}
	defer monitor.Stop()

	callCtx := engine.NewContext(ctx, &prg)
	return r.call(callCtx, monitor, env, input)
}

type Event struct {
	Time    time.Time
	Context *engine.Context
	Type    EventType
	Debug   any
	Content string
}

type EventType string

var (
	EventTypeStart  = EventType("start")
	EventTypeUpdate = EventType("progress")
	EventTypeDebug  = EventType("debug")
	EventTypeStop   = EventType("stop")
)

func (r *Runner) call(callCtx engine.Context, monitor Monitor, env []string, input string) (string, error) {
	progress := make(chan openai.Status)

	e := engine.Engine{
		Client:   r.c,
		Progress: progress,
		Env:      env,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range progress {
			if message := status.PartialResponse; message != nil {
				content := message.String()
				if strings.TrimSpace(content) != "" && !strings.Contains(content, "Sent content:") {
					content += "\n\nModel responding..."
				}
				monitor.Event(Event{
					Time:    time.Now(),
					Context: &callCtx,
					Type:    EventTypeUpdate,
					Content: content,
				})
			} else {
				monitor.Event(Event{
					Time:    time.Now(),
					Context: &callCtx,
					Type:    EventTypeDebug,
					Debug:   status,
				})
			}
		}
	}()
	defer wg.Wait()
	defer close(progress)

	monitor.Event(Event{
		Time:    time.Now(),
		Context: &callCtx,
		Type:    EventTypeStart,
		Content: input,
	})

	result, err := e.Start(callCtx, input)
	if err != nil {
		return "", err
	}

	for {
		if result.Result != nil && len(result.Calls) == 0 {
			monitor.Event(Event{
				Time:    time.Now(),
				Context: &callCtx,
				Type:    EventTypeStop,
				Content: *result.Result,
			})
			return *result.Result, nil
		}

		var (
			callResults []engine.CallResult
			resultLock  sync.Mutex
		)

		eg, subCtx := errgroup.WithContext(callCtx.Ctx)
		for id, call := range result.Calls {
			id := id
			call := call
			eg.Go(func() error {
				callCtx, err := callCtx.SubCall(subCtx, call.ToolName, id)
				if err != nil {
					return err
				}

				result, err := r.call(callCtx, monitor, env, call.Input)
				if err != nil {
					return err
				}

				resultLock.Lock()
				defer resultLock.Unlock()
				callResults = append(callResults, engine.CallResult{
					ID:     id,
					Result: result,
				})

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return "", err
		}

		result, err = e.Continue(callCtx.Ctx, result.State, callResults...)
		if err != nil {
			return "", err
		}
	}
}
