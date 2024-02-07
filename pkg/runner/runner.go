package runner

import (
	"context"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"golang.org/x/sync/errgroup"
)

type MonitorFactory interface {
	Start(ctx context.Context, prg *types.Program, env []string, input string) (Monitor, error)
}

type Monitor interface {
	Event(event Event)
	Stop(output string, err error)
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

func (r *Runner) Run(ctx context.Context, prg types.Program, env []string, input string) (output string, err error) {
	monitor, err := r.factory.Start(ctx, &prg, env, input)
	if err != nil {
		return "", err
	}
	defer func() {
		monitor.Stop(output, err)
	}()

	callCtx := engine.NewContext(ctx, &prg)
	return r.call(callCtx, monitor, env, input)
}

type Event struct {
	Time               time.Time       `json:"time,omitempty"`
	CallContext        *engine.Context `json:"callContext,omitempty"`
	ToolResults        int             `json:"toolResults,omitempty"`
	Type               EventType       `json:"type,omitempty"`
	ChatCompletionID   string          `json:"chatCompletionId,omitempty"`
	ChatRequest        any             `json:"chatRequest,omitempty"`
	ChatResponse       any             `json:"chatResponse,omitempty"`
	ChatResponseCached bool            `json:"chatResponseCached,omitempty"`
	Content            string          `json:"content,omitempty"`
}

type EventType string

var (
	EventTypeCallStart    = EventType("callStart")
	EventTypeCallContinue = EventType("callContinue")
	EventTypeCallProgress = EventType("callProgress")
	EventTypeChat         = EventType("callChat")
	EventTypeCallFinish   = EventType("callFinish")
)

func (r *Runner) call(callCtx engine.Context, monitor Monitor, env []string, input string) (string, error) {
	progress, progressClose := streamProgress(&callCtx, monitor)
	defer progressClose()

	e := engine.Engine{
		Client:   r.c,
		Progress: progress,
		Env:      env,
	}

	monitor.Event(Event{
		Time:        time.Now(),
		CallContext: &callCtx,
		Type:        EventTypeCallStart,
		Content:     input,
	})

	result, err := e.Start(callCtx, input)
	if err != nil {
		return "", err
	}

	for {
		if result.Result != nil && len(result.Calls) == 0 {
			progressClose()
			monitor.Event(Event{
				Time:        time.Now(),
				CallContext: &callCtx,
				Type:        EventTypeCallFinish,
				Content:     *result.Result,
			})
			return *result.Result, nil
		}

		callResults, err := r.subCalls(callCtx, monitor, env, result)
		if err != nil {
			return "", err
		}

		monitor.Event(Event{
			Time:        time.Now(),
			CallContext: &callCtx,
			Type:        EventTypeCallContinue,
			ToolResults: len(callResults),
		})

		result, err = e.Continue(callCtx.Ctx, result.State, callResults...)
		if err != nil {
			return "", err
		}
	}
}

func streamProgress(callCtx *engine.Context, monitor Monitor) (chan openai.Status, func()) {
	progress := make(chan openai.Status)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range progress {
			if message := status.PartialResponse; message != nil {
				monitor.Event(Event{
					Time:             time.Now(),
					CallContext:      callCtx,
					Type:             EventTypeCallProgress,
					ChatCompletionID: status.CompletionID,
					Content:          message.String(),
				})
			} else {
				monitor.Event(Event{
					Time:               time.Now(),
					CallContext:        callCtx,
					Type:               EventTypeChat,
					ChatCompletionID:   status.CompletionID,
					ChatRequest:        status.Request,
					ChatResponse:       status.Response,
					ChatResponseCached: status.Cached,
				})
			}
		}
	}()

	var once sync.Once
	return progress, func() {
		once.Do(func() {
			close(progress)
			wg.Wait()
		})
	}
}

func (r *Runner) subCalls(callCtx engine.Context, monitor Monitor, env []string, lastReturn *engine.Return) (callResults []engine.CallResult, _ error) {
	var (
		resultLock sync.Mutex
	)

	eg, subCtx := errgroup.WithContext(callCtx.Ctx)
	for id, call := range lastReturn.Calls {
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
		return nil, err
	}

	return
}
