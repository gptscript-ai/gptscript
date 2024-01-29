package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/acorn-io/gptscript/pkg/cache"
	"github.com/acorn-io/gptscript/pkg/engine"
	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/types"
	"github.com/pterm/pterm"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	Quiet bool
	c     *openai.Client
	mw    pterm.MultiPrinter
}

func New() (*Runner, error) {
	cache, err := cache.New()
	if err != nil {
		return nil, err
	}
	c, err := openai.NewClient(cache)
	if err != nil {
		return nil, err
	}
	return &Runner{
		c:  c,
		mw: pterm.DefaultMultiPrinter,
	}, nil
}

func (r *Runner) Run(ctx context.Context, tool types.Tool, toolSet types.ToolSet, input string) (string, error) {
	progress := make(chan types.CompletionMessage)
	go func() {
		for next := range progress {
			fmt.Print(next.String())
		}
		fmt.Println()
	}()

	if !r.Quiet {
		r.mw.Start()
		defer func() {
			r.mw.Stop()
			fmt.Println()
		}()
	}

	return r.call(ctx, nil, tool, toolSet, input)
}

func prefix(parent *engine.Context) string {
	if parent == nil {
		return ""
	}
	return ".." + prefix(parent.Parent)
}

func infoMsg(prefix string, tool types.Tool, input, output string) string {
	if len(input) > 20 {
		input = input[:20]
	}
	prefix = fmt.Sprintf("%sCall (%s): %s -> ", prefix, tool.Name, input)
	maxLen := pterm.GetTerminalWidth() - len(prefix) - 8
	str := fmt.Sprintf("%s", output)
	if len(str) > maxLen && maxLen > 0 {
		str = str[len(str)-maxLen:]
	}

	return strings.ReplaceAll(prefix+str, "\n", " ")
}

func (r *Runner) call(ctx context.Context, parent *engine.Context, tool types.Tool, toolSet types.ToolSet, input string) (string, error) {
	prefix := prefix(parent)
	spinner, err := pterm.DefaultSpinner.WithWriter(r.mw.NewWriter()).Start(fmt.Sprintf("%sCall (%s): %s", prefix, tool.Name, input))
	if err != nil {
		return "", err
	}
	defer spinner.Stop()

	progress := make(chan types.CompletionMessage)
	progressBuffer := &strings.Builder{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		for message := range progress {
			progressBuffer.WriteString(message.String())
			spinner.Info(infoMsg(prefix, tool, input, progressBuffer.String()))
		}
	}()
	defer close(progress)

	e := engine.Engine{
		Client:   r.c,
		Progress: progress,
	}

	callCtx := engine.Context{
		Ctx:    ctx,
		Parent: parent,
		Tool:   tool,
	}
	for _, toolName := range tool.Tools {
		callCtx.Tools = append(callCtx.Tools, toolSet[toolName])
	}

	result, err := e.Start(callCtx, input)
	if err != nil {
		return "", err
	}

	for {
		if result.Result != nil {
			spinner.Info(infoMsg(prefix, tool, input, *result.Result))
			return *result.Result, nil
		}

		var (
			callResults []engine.CallResult
			resultLock  sync.Mutex
		)

		eg, subCtx := errgroup.WithContext(ctx)
		for id, call := range result.Calls {
			id := id
			call := call
			tool := toolSet[call.ToolName]
			eg.Go(func() error {
				result, err := r.call(subCtx, &callCtx, tool, toolSet, call.Input)
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

		result, err = e.Continue(ctx, result.State, callResults...)
		if err != nil {
			return "", err
		}
	}
}
