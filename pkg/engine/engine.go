package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/types"
	"github.com/acorn-io/gptscript/pkg/version"
)

type ErrToolNotFound struct {
	ToolName string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

type Engine struct {
	Client   *openai.Client
	Progress chan<- types.CompletionMessage
}

// +k8s:deepcopy-gen=true

type State struct {
	Completion types.CompletionRequest             `json:"completion,omitempty"`
	Pending    map[string]types.CompletionToolCall `json:"pending,omitempty"`
	Results    map[string]CallResult               `json:"results,omitempty"`
}

type Return struct {
	State  *State
	Calls  map[string]Call
	Result *string
}

type Call struct {
	ToolName string `json:"toolName,omitempty"`
	Input    string `json:"input,omitempty"`
}

type CallResult struct {
	ID     string `json:"id,omitempty"`
	Result string `json:"result,omitempty"`
}

type Context struct {
	ID     string                `json:"id,omitempty"`
	Ctx    context.Context       `json:"-"`
	Parent *Context              `json:"parent,omitempty"`
	Tool   types.Tool            `json:"tool,omitempty"`
	Tools  map[string]types.Tool `json:"tools,omitempty"`
}

var execID int32

func NewContext(ctx context.Context, parent *Context, tool types.Tool, tools map[string]types.Tool) Context {
	callCtx := Context{
		ID:     fmt.Sprint(atomic.AddInt32(&execID, 1)),
		Ctx:    ctx,
		Parent: parent,
		Tool:   tool,
		Tools:  tools,
	}
	return callCtx
}

func (c *Context) getTool(name string) (types.Tool, error) {
	for _, tool := range c.Tools {
		if tool.Name == name {
			return tool, nil
		}
	}

	return types.Tool{}, &ErrToolNotFound{
		ToolName: name,
	}
}

func (e *Engine) runCommand(tool types.Tool, input string) (string, error) {
	env := os.Environ()
	data := map[string]any{}

	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()

	if err := json.Unmarshal([]byte(input), &data); err == nil {
		for k, v := range data {
			envName := fmt.Sprintf("ARG_%s", strings.ToUpper(strings.ReplaceAll(k, "-", "_")))
			switch val := v.(type) {
			case string:
				env = append(env, envName+"="+val)
			case json.Number:
				env = append(env, envName+"="+string(val))
			case bool:
				env = append(env, envName+"="+fmt.Sprint(val))
			default:
				data, err := json.Marshal(val)
				if err == nil {
					env = append(env, envName+"="+string(data))
				}
			}
		}
	}

	interpreter, rest, _ := strings.Cut(tool.Instructions, "\n")
	f, err := os.CreateTemp("", version.ProgramName)
	if err != nil {
		return "", err
	}

	_, err = f.Write([]byte(rest))
	_ = f.Close()
	if err != nil {
		return "", err
	}
	interpreter = strings.TrimSpace(interpreter)[2:]
	output := &bytes.Buffer{}

	cmd := exec.Command(interpreter, f.Name())
	cmd.Env = env
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	cmd.Stdout = output

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return string(output.Bytes()), nil
}

func (e *Engine) Start(ctx Context, input string) (*Return, error) {
	tool := ctx.Tool

	if tool.IsCommand() {
		s, err := e.runCommand(tool, input)
		if err != nil {
			return nil, err
		}
		return &Return{
			Result: &s,
		}, nil
	}

	completion := types.CompletionRequest{
		Model:        tool.ModelName,
		Vision:       tool.Vision,
		Tools:        nil,
		Messages:     nil,
		MaxToken:     tool.MaxTokens,
		JSONResponse: tool.JSONResponse,
		Cache:        tool.Cache,
	}

	for _, subToolName := range tool.Tools {
		subTool, err := ctx.getTool(subToolName)
		if err != nil {
			return nil, err
		}
		completion.Tools = append(completion.Tools, types.CompletionTool{
			Type: types.CompletionToolTypeFunction,
			Function: types.CompletionFunctionDefinition{
				Name:        subToolName,
				Description: subTool.Description,
				Parameters:  subTool.Arguments,
			},
		})
	}

	if tool.Instructions != "" {
		completion.Messages = append(completion.Messages, types.CompletionMessage{
			Role: types.CompletionMessageRoleTypeSystem,
			Content: []types.ContentPart{
				{
					Text: tool.Instructions,
				},
			},
		})
	}

	if input != "" {
		completion.Messages = append(completion.Messages, types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeUser,
			Content: types.Text(input),
		})
	}

	return e.complete(ctx.Ctx, &State{
		Completion: completion,
	})
}

func (e *Engine) complete(ctx context.Context, state *State) (*Return, error) {
	var (
		progress = make(chan types.CompletionMessage)
		ret      = Return{
			State: state,
			Calls: map[string]Call{},
		}
		wg sync.WaitGroup
	)

	// ensure we aren't writing to the channel anymore on exit
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		for message := range progress {
			if e.Progress != nil {
				e.Progress <- message
			}
		}
	}()

	resp, err := e.Client.Call(ctx, state.Completion, progress)
	close(progress)
	if err != nil {
		return nil, err
	}

	state.Completion.Messages = append(state.Completion.Messages, *resp)

	state.Pending = map[string]types.CompletionToolCall{}
	for _, content := range resp.Content {
		if content.ToolCall != nil {
			state.Pending[content.ToolCall.ID] = *content.ToolCall
			ret.Calls[content.ToolCall.ID] = Call{
				ToolName: content.ToolCall.Function.Name,
				Input:    content.ToolCall.Function.Arguments,
			}
		}
		if content.Text != "" {
			cp := content.Text
			ret.Result = &cp
		}
	}

	return &ret, nil
}

func (e *Engine) Continue(ctx context.Context, state *State, results ...CallResult) (*Return, error) {
	state = state.DeepCopy()

	if state.Results == nil {
		state.Results = map[string]CallResult{}
	}

	for _, result := range results {
		state.Results[result.ID] = result
	}

	ret := Return{
		State: state,
		Calls: map[string]Call{},
	}

	var (
		added            bool
		pendingToolCalls []types.CompletionToolCall
	)

	for id, pending := range state.Pending {
		pendingToolCalls = append(pendingToolCalls, pending)
		if _, ok := state.Results[id]; !ok {
			ret.Calls[id] = Call{
				ToolName: pending.Function.Name,
				Input:    pending.Function.Arguments,
			}
		}
	}

	if len(ret.Calls) > 0 {
		return &ret, nil
	}

	sort.Slice(pendingToolCalls, func(i, j int) bool {
		left := pendingToolCalls[i].Function.Name + pendingToolCalls[i].Function.Arguments
		right := pendingToolCalls[j].Function.Name + pendingToolCalls[j].Function.Arguments
		if left == right {
			return pendingToolCalls[i].ID < pendingToolCalls[j].ID
		}
		return left < right
	})

	for _, pending := range pendingToolCalls {
		pending := pending
		if result, ok := state.Results[pending.ID]; ok {
			added = true
			state.Completion.Messages = append(state.Completion.Messages, types.CompletionMessage{
				Role:     types.CompletionMessageRoleTypeTool,
				Content:  types.Text(result.Result),
				ToolCall: &pending,
			})
		}
	}

	if !added {
		return nil, fmt.Errorf("invalid continue call, no completion needed")
	}

	return e.complete(ctx, state)
}
