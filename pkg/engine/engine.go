package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gptscript-ai/gptscript/pkg/system"
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
	State  *State          `json:"state,omitempty"`
	Calls  map[string]Call `json:"calls,omitempty"`
	Result *string         `json:"result,omitempty"`
}

type Call struct {
	ToolID string `json:"toolID,omitempty"`
	Input  string `json:"input,omitempty"`
}

type CallResult struct {
	ToolID string `json:"toolID,omitempty"`
	CallID string `json:"callID,omitempty"`
	Result string `json:"result,omitempty"`
	User   string `json:"user,omitempty"`
}

type commonContext struct {
	ID           string         `json:"id"`
	Tool         types.Tool     `json:"tool"`
	InputContext []InputContext `json:"inputContext"`
	ToolCategory ToolCategory   `json:"toolCategory,omitempty"`
}

type CallContext struct {
	commonContext `json:",inline"`
	ToolName      string `json:"toolName,omitempty"`
	ParentID      string `json:"parentID,omitempty"`
}

type Context struct {
	commonContext
	Ctx          context.Context
	Parent       *Context
	Program      *types.Program
	ToolCategory ToolCategory
}

type ToolCategory string

const (
	CredentialToolCategory ToolCategory = "credential"
	ContextToolCategory    ToolCategory = "context"
	NoCategory             ToolCategory = ""
)

type InputContext struct {
	ToolID  string `json:"toolID,omitempty"`
	Content string `json:"content,omitempty"`
}

func (c *Context) ParentID() string {
	if c.Parent == nil {
		return ""
	}
	return c.Parent.ID
}

func (c *Context) GetCallContext() *CallContext {
	var toolName string
	if c.Parent != nil {
		for name, id := range c.Parent.Tool.ToolMapping {
			if id == c.Tool.ID {
				toolName = name
				break
			}
		}
	}

	return &CallContext{
		commonContext: c.commonContext,
		ParentID:      c.ParentID(),
		ToolName:      toolName,
	}
}

func (c *Context) UnmarshalJSON([]byte) error {
	panic("this data struct is circular by design and can not be read from json")
}

func (c *Context) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.GetCallContext())
}

var execID int32

func NewContext(ctx context.Context, prg *types.Program) Context {
	callCtx := Context{
		commonContext: commonContext{
			ID:   fmt.Sprint(atomic.AddInt32(&execID, 1)),
			Tool: prg.ToolSet[prg.EntryToolID],
		},
		Ctx:     ctx,
		Program: prg,
	}
	return callCtx
}

func (c *Context) SubCall(ctx context.Context, toolID, callID string, toolCategory ToolCategory) (Context, error) {
	tool, ok := c.Program.ToolSet[toolID]
	if !ok {
		return Context{}, fmt.Errorf("failed to file tool for id [%s]", toolID)
	}

	if callID == "" {
		callID = fmt.Sprint(atomic.AddInt32(&execID, 1))
	}

	return Context{
		commonContext: commonContext{
			ID:           callID,
			Tool:         tool,
			ToolCategory: toolCategory,
		},
		Ctx:     ctx,
		Parent:  c,
		Program: c.Program,
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
		} else if tool.IsPrint() {
			return e.runPrint(tool)
		}
		s, err := e.runCommand(ctx.WrappedContext(), tool, input, ctx.ToolCategory)
		if err != nil {
			return nil, err
		}
		return &Return{
			Result: &s,
		}, nil
	}

	if ctx.ToolCategory == CredentialToolCategory {
		return nil, fmt.Errorf("credential tools cannot make calls to the LLM")
	}

	completion := types.CompletionRequest{
		Model:                tool.Parameters.ModelName,
		MaxTokens:            tool.Parameters.MaxTokens,
		JSONResponse:         tool.Parameters.JSONResponse,
		Cache:                tool.Parameters.Cache,
		Temperature:          tool.Parameters.Temperature,
		InternalSystemPrompt: tool.Parameters.InternalPrompt,
	}

	if tool.Chat && completion.InternalSystemPrompt == nil {
		completion.InternalSystemPrompt = new(bool)
	}

	var err error
	completion.Tools, err = tool.GetCompletionTools(*ctx.Program)
	if err != nil {
		return nil, err
	}

	completion.Messages = addUpdateSystem(ctx, tool, completion.Messages)

	if _, def := system.IsDefaultPrompt(input); tool.Chat && def {
		// Ignore "default prompts" from chat
		input = ""
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

func addUpdateSystem(ctx Context, tool types.Tool, msgs []types.CompletionMessage) []types.CompletionMessage {
	var instructions []string

	for _, context := range ctx.InputContext {
		instructions = append(instructions, context.Content)
	}

	if tool.Instructions != "" {
		instructions = append(instructions, tool.Instructions)
	}

	if len(instructions) == 0 {
		return msgs
	}

	msg := types.CompletionMessage{
		Role:    types.CompletionMessageRoleTypeSystem,
		Content: types.Text(strings.Join(instructions, "\n")),
	}

	if len(msgs) > 0 && msgs[0].Role == types.CompletionMessageRoleTypeSystem {
		return append([]types.CompletionMessage{msg}, msgs[1:]...)
	}

	return append([]types.CompletionMessage{msg}, msgs...)
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

func (e *Engine) Continue(ctx Context, state *State, results ...CallResult) (*Return, error) {
	var added bool

	state = &State{
		Completion: state.Completion,
		Pending:    state.Pending,
		Results:    map[string]CallResult{},
	}

	for _, result := range results {
		if result.CallID != "" {
			state.Results[result.CallID] = result
		}
		if result.User != "" {
			added = true
			state.Completion.Messages = append(state.Completion.Messages, types.CompletionMessage{
				Role:    types.CompletionMessageRoleTypeUser,
				Content: types.Text(result.User),
			})
		}
	}

	ret := Return{
		State: state,
		Calls: map[string]Call{},
	}

	for id, pending := range state.Pending {
		if _, ok := state.Results[id]; !ok {
			ret.Calls[id] = Call{
				ToolID: state.Completion.Tools[*pending.Index].Function.ToolID,
				Input:  pending.Function.Arguments,
			}
		}
	}

	if len(ret.Calls) > 0 {
		// Outstanding tool calls still pending
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

	state.Completion.Messages = addUpdateSystem(ctx, ctx.Tool, state.Completion.Messages)
	return e.complete(ctx.Ctx, state)
}
