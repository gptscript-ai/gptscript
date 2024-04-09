package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

var completionID int64

type Model interface {
	Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error)
}

type RuntimeManager interface {
	GetContext(ctx context.Context, tool types.Tool, cmd, env []string) (string, []string, error)
}

type Engine struct {
	Model          Model
	RuntimeManager RuntimeManager
	Env            []string
	Progress       chan<- types.CompletionStatus
	Ports          *Ports
}

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
	ToolID string `json:"toolID,omitempty"`
	Input  string `json:"input,omitempty"`
}

type CallResult struct {
	ToolID string `json:"toolID,omitempty"`
	CallID string `json:"callID,omitempty"`
	Result string `json:"result,omitempty"`
}

type Context struct {
	ID           string
	Ctx          context.Context
	Parent       *Context
	Program      *types.Program
	Tool         types.Tool
	InputContext []InputContext
}

type InputContext struct {
	ToolName string `json:"toolName,omitempty"`
	Content  string `json:"content,omitempty"`
}

func (c *Context) ParentID() string {
	if c.Parent == nil {
		return ""
	}
	return c.Parent.ID
}

func (c *Context) UnmarshalJSON([]byte) error {
	panic("this data struct is circular by design and can not be read from json")
}

func (c *Context) MarshalJSON() ([]byte, error) {
	var parentID string
	if c.Parent != nil {
		parentID = c.Parent.ID
	}
	return json.Marshal(map[string]any{
		"id":           c.ID,
		"parentID":     parentID,
		"tool":         c.Tool,
		"inputContext": c.InputContext,
	})
}

var execID int32

func NewContext(ctx context.Context, prg *types.Program) Context {
	callCtx := Context{
		ID:      fmt.Sprint(atomic.AddInt32(&execID, 1)),
		Ctx:     ctx,
		Program: prg,
		Tool:    prg.ToolSet[prg.EntryToolID],
	}
	return callCtx
}

func (c *Context) SubCall(ctx context.Context, toolID, callID string) (Context, error) {
	tool, ok := c.Program.ToolSet[toolID]
	if !ok {
		return Context{}, fmt.Errorf("failed to file tool for id [%s]", toolID)
	}

	if callID == "" {
		callID = fmt.Sprint(atomic.AddInt32(&execID, 1))
	}

	return Context{
		ID:      callID,
		Ctx:     ctx,
		Parent:  c,
		Program: c.Program,
		Tool:    tool,
	}, nil
}

type engineContext struct{}

func FromContext(ctx context.Context) (*Context, bool) {
	c, ok := ctx.Value(engineContext{}).(*Context)
	return c, ok
}

func (c *Context) WrappedContext() context.Context {
	return context.WithValue(c.Ctx, engineContext{}, c)
}

func (e *Engine) Start(ctx Context, input string) (*Return, error) {
	tool := ctx.Tool

	if tool.IsCommand() {
		if tool.IsHTTP() {
			return e.runHTTP(ctx.Ctx, ctx.Program, tool, input)
		} else if tool.IsDaemon() {
			return e.runDaemon(ctx.Ctx, ctx.Program, tool, input)
		} else if tool.IsOpenAPI() {
			return e.runOpenAPI(tool, input)
		}
		s, err := e.runCommand(ctx.WrappedContext(), tool, input)
		if err != nil {
			return nil, err
		}
		return &Return{
			Result: &s,
		}, nil
	}

	completion := types.CompletionRequest{
		Model:                tool.Parameters.ModelName,
		MaxTokens:            tool.Parameters.MaxTokens,
		JSONResponse:         tool.Parameters.JSONResponse,
		Cache:                tool.Parameters.Cache,
		Temperature:          tool.Parameters.Temperature,
		InternalSystemPrompt: tool.Parameters.InternalPrompt,
	}

	var err error
	completion.Tools, err = tool.GetCompletionTools(*ctx.Program)
	if err != nil {
		return nil, err
	}

	var instructions []string

	for _, context := range ctx.InputContext {
		instructions = append(instructions, context.Content)
	}

	if tool.Instructions != "" {
		instructions = append(instructions, tool.Instructions)
	}

	if len(instructions) > 0 {
		completion.Messages = append(completion.Messages, types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeSystem,
			Content: types.Text(strings.Join(instructions, "\n")),
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
		progress = make(chan types.CompletionStatus)
		ret      = Return{
			State: state,
			Calls: map[string]Call{},
		}
		wg sync.WaitGroup
	)

	// ensure we aren't writing to the channel anymore on exit
	wg.Add(1)
	defer wg.Wait()
	defer close(progress)

	go func() {
		defer wg.Done()
		for message := range progress {
			if e.Progress != nil {
				e.Progress <- message
			}
		}
	}()

	resp, err := e.Model.Call(ctx, state.Completion, progress)
	if err != nil {
		return nil, err
	}

	state.Completion.Messages = append(state.Completion.Messages, *resp)

	state.Pending = map[string]types.CompletionToolCall{}
	for _, content := range resp.Content {
		if content.ToolCall != nil {
			var toolID string
			for _, tool := range state.Completion.Tools {
				if tool.Function.Name == content.ToolCall.Function.Name {
					toolID = tool.Function.ToolID
				}
			}
			if toolID == "" {
				return nil, fmt.Errorf("failed to find tool id for tool %s in tool_call result", content.ToolCall.Function.Name)
			}
			state.Pending[content.ToolCall.ID] = *content.ToolCall
			ret.Calls[content.ToolCall.ID] = Call{
				ToolID: toolID,
				Input:  content.ToolCall.Function.Arguments,
			}
		} else {
			cp := content.Text
			ret.Result = &cp
		}
	}

	return &ret, nil
}

func (e *Engine) Continue(ctx context.Context, state *State, results ...CallResult) (*Return, error) {
	state = &State{
		Completion: state.Completion,
		Pending:    state.Pending,
		Results:    map[string]CallResult{},
	}

	for _, result := range results {
		state.Results[result.CallID] = result
	}

	ret := Return{
		State: state,
		Calls: map[string]Call{},
	}

	var (
		added bool
	)

	for id, pending := range state.Pending {
		if _, ok := state.Results[id]; !ok {
			ret.Calls[id] = Call{
				ToolID: state.Completion.Tools[*pending.Index].Function.ToolID,
				Input:  pending.Function.Arguments,
			}
		}
	}

	if len(ret.Calls) > 0 {
		return &ret, nil
	}

	for _, content := range state.Completion.Messages[len(state.Completion.Messages)-1].Content {
		if content.ToolCall == nil {
			continue
		}
		result, ok := state.Results[content.ToolCall.ID]
		if !ok {
			return nil, fmt.Errorf("missing tool call result for id %s, most likely a %s BUG",
				content.ToolCall.ID, version.ProgramName)
		}

		pending, ok := state.Pending[content.ToolCall.ID]
		if !ok {
			return nil, fmt.Errorf("missing tool call pending for id %s, most likely a %s BUG",
				content.ToolCall.ID, version.ProgramName)
		}

		added = true
		state.Completion.Messages = append(state.Completion.Messages, types.CompletionMessage{
			Role:     types.CompletionMessageRoleTypeTool,
			Content:  types.Text(result.Result),
			ToolCall: &pending,
		})
	}

	if !added {
		return nil, fmt.Errorf("invalid continue call, no completion needed")
	}

	return e.complete(ctx, state)
}
