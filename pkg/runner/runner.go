package runner

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/gptscript/pkg/cache"
	"github.com/acorn-io/gptscript/pkg/engine"
	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/types"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	Quiet        bool   `usage:"Do not print status" short:"q"`
	DumpState    string `usage:"Dump the internal execution state to a file"`
	Cache        *bool  `usage:"Disable caching" default:"true"`
	ShowFinished bool   `usage:"Show finished calls results"`
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		if opt.DumpState != "" {
			result.DumpState = opt.DumpState
		}
		if opt.Quiet {
			result.Quiet = true
		}
		if opt.Cache != nil {
			result.Cache = opt.Cache
		}
		if opt.ShowFinished {
			result.ShowFinished = opt.ShowFinished
		}
	}
	if result.Cache == nil {
		result.Cache = &[]bool{true}[0]
	}
	return
}

type Runner struct {
	Quiet     bool
	c         *openai.Client
	display   *display
	dumpState string
}

func New(opts ...Options) (*Runner, error) {
	opt := complete(opts...)

	cacheBackend := cache.NoCache()
	if *opt.Cache {
		var err error
		cacheBackend, err = cache.New()
		if err != nil {
			return nil, err
		}
	}

	c, err := openai.NewClient(cacheBackend)
	if err != nil {
		return nil, err
	}
	return &Runner{
		c:         c,
		display:   newDisplay(opt.Quiet, opt.ShowFinished),
		dumpState: opt.DumpState,
	}, nil
}

func (r *Runner) Run(ctx context.Context, tool types.Tool, input string) (string, error) {
	if err := r.display.Start(ctx); err != nil {
		return "", err
	}

	defer func() {
		_ = r.display.Stop()
		if r.dumpState != "" {
			f, err := os.Create(r.dumpState)
			if err == nil {
				r.display.Dump(f)
				f.Close()
			}
		}
	}()

	callCtx := engine.NewContext(ctx, nil, tool)
	return r.call(callCtx, input)
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

func (r *Runner) call(callCtx engine.Context, input string) (string, error) {
	progress := make(chan openai.Status)

	e := engine.Engine{
		Client:   r.c,
		Progress: progress,
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
				r.display.progress <- Event{
					Time:    time.Now(),
					Context: &callCtx,
					Type:    EventTypeUpdate,
					Content: content,
				}
			} else {
				r.display.progress <- Event{
					Time:    time.Now(),
					Context: &callCtx,
					Type:    EventTypeDebug,
					Debug:   status,
				}
			}
		}
	}()
	defer wg.Wait()
	defer close(progress)

	r.display.progress <- Event{
		Time:    time.Now(),
		Context: &callCtx,
		Type:    EventTypeStart,
		Content: input,
	}

	result, err := e.Start(callCtx, input)
	if err != nil {
		return "", err
	}

	for {
		if result.Result != nil {
			r.display.progress <- Event{
				Time:    time.Now(),
				Context: &callCtx,
				Type:    EventTypeStop,
				Content: *result.Result,
			}
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
			callCtx := engine.NewContext(subCtx, &callCtx, callCtx.Tool.ToolSet[call.ToolName])
			callCtx.ID = id
			eg.Go(func() error {
				result, err := r.call(callCtx, call.Input)
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
