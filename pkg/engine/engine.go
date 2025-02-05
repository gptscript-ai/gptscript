package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/config"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

type Model interface {
	Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error)
	ProxyInfo() (string, string, error)
}

type RuntimeManager interface {
	GetContext(ctx context.Context, tool types.Tool, cmd, env []string) (string, []string, error)
	EnsureCredentialHelpers(ctx context.Context) error
	SetUpCredentialHelpers(ctx context.Context, cliCfg *config.CLIConfig) error
}

type Engine struct {
	Model          Model
	RuntimeManager RuntimeManager
	Env            []string
	Progress       chan<- types.CompletionStatus
}

type State struct {
	Input      string                              `json:"input,omitempty"`
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
	Missing bool   `json:"missing,omitempty"`
	ToolID  string `json:"toolID,omitempty"`
	Input   string `json:"input,omitempty"`
}

type CallResult struct {
	ToolID string `json:"toolID,omitempty"`
	CallID string `json:"callID,omitempty"`
	Result string `json:"result,omitempty"`
	User   string `json:"user,omitempty"`
}

type commonContext struct {
	ID           string                `json:"id"`
	Tool         types.Tool            `json:"tool"`
	CurrentAgent types.ToolReference   `json:"currentAgent,omitempty"`
	AgentGroup   []types.ToolReference `json:"agentGroup,omitempty"`
	InputContext []InputContext        `json:"inputContext"`
	ToolCategory ToolCategory          `json:"toolCategory,omitempty"`
}

type CallContext struct {
	commonContext `json:",inline"`
	ToolName      string `json:"toolName,omitempty"`
	ParentID      string `json:"parentID,omitempty"`
	DisplayText   string `json:"displayText,omitempty"`
}

type Context struct {
	commonContext
	Ctx           context.Context
	Parent        *Context
	LastReturn    *Return
	CurrentReturn *Return
	Engine        *Engine
	Program       *types.Program
	// Input is saved only so that we can render display text, don't use otherwise
	Input string
}

type ChatHistory struct {
	History []ChatHistoryCall `json:"history,omitempty"`
}

type ChatHistoryCall struct {
	ID         string                  `json:"id,omitempty"`
	Tool       types.Tool              `json:"tool,omitempty"`
	Completion types.CompletionRequest `json:"completion,omitempty"`
}

type ToolCategory string

const (
	ProviderToolCategory   ToolCategory = "provider"
	CredentialToolCategory ToolCategory = "credential"
	ContextToolCategory    ToolCategory = "context"
	InputToolCategory      ToolCategory = "input"
	OutputToolCategory     ToolCategory = "output"
	NoCategory             ToolCategory = ""
)

type InputContext struct {
	ToolID  string `json:"toolID,omitempty"`
	Content string `json:"content,omitempty"`
}

type ErrChatFinish struct {
	Message string
}

func (e *ErrChatFinish) Error() string {
	return fmt.Sprintf("CHAT FINISH: %s", e.Message)
}

func IsChatFinishMessage(msg string) error {
	if msg, ok := strings.CutPrefix(msg, "CHAT FINISH: "); ok {
		return &ErrChatFinish{Message: msg}
	}
	return nil
}

func (c *Context) ParentID() string {
	if c.Parent == nil {
		return ""
	}
	return c.Parent.ID
}

func (c *Context) CurrentAgent() types.ToolReference {
	for _, ref := range c.AgentGroup {
		if ref.ToolID == c.Tool.ID {
			return ref
		}
	}
	if c.Parent != nil {
		return c.Parent.CurrentAgent()
	}
	return types.ToolReference{}
}

func (c *Context) GetCallContext() *CallContext {
	var toolName string
	if c.Parent != nil {
	outer:
		for name, refs := range c.Parent.Tool.ToolMapping {
			for _, ref := range refs {
				if ref.ToolID == c.Tool.ID {
					toolName = name
					break outer
				}
			}
		}
	}

	result := &CallContext{
		commonContext: c.commonContext,
		ParentID:      c.ParentID(),
		ToolName:      toolName,
		DisplayText:   types.ToDisplayText(c.Tool, c.Input),
	}

	result.CurrentAgent = c.CurrentAgent()
	return result
}

func (c *Context) UnmarshalJSON([]byte) error {
	panic("this data struct is circular by design and can not be read from json")
}

func (c *Context) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.GetCallContext())
}

type toolCategoryKey struct{}

func WithToolCategory(ctx context.Context, toolCategory ToolCategory) context.Context {
	return context.WithValue(ctx, toolCategoryKey{}, toolCategory)
}

func ToolCategoryFromContext(ctx context.Context) ToolCategory {
	category, _ := ctx.Value(toolCategoryKey{}).(ToolCategory)
	return category
}

func NewContext(ctx context.Context, prg *types.Program, input string) (Context, error) {
	category := ToolCategoryFromContext(ctx)

	callCtx := Context{
		commonContext: commonContext{
			ID:           counter.Next(),
			Tool:         prg.ToolSet[prg.EntryToolID],
			ToolCategory: category,
		},
		Ctx:     ctx,
		Program: prg,
		Input:   input,
	}

	agentGroup, err := callCtx.Tool.GetToolsByType(prg, types.ToolTypeAgent)
	if err != nil {
		return callCtx, err
	}

	callCtx.AgentGroup = agentGroup

	if callCtx.Tool.IsAgentsOnly() && len(callCtx.AgentGroup) > 0 {
		callCtx.Tool = callCtx.Program.ToolSet[callCtx.AgentGroup[0].ToolID]
	}

	return callCtx, nil
}

func (c *Context) SubCallContext(ctx context.Context, input, toolID, callID string, toolCategory ToolCategory) (Context, error) {
	tool := c.Program.ToolSet[toolID]

	if callID == "" {
		callID = counter.Next()
	}

	agentGroup, err := c.Tool.GetNextAgentGroup(c.Program, c.AgentGroup, toolID)
	if err != nil {
		return Context{}, err
	}

	return Context{
		commonContext: commonContext{
			ID:           callID,
			Tool:         tool,
			AgentGroup:   agentGroup,
			ToolCategory: toolCategory,
		},
		Ctx:           ctx,
		Parent:        c,
		Program:       c.Program,
		CurrentReturn: c.CurrentReturn,
		Input:         input,
	}, nil
}

type engineContext struct{}

func FromContext(ctx context.Context) (*Context, bool) {
	c, ok := ctx.Value(engineContext{}).(*Context)
	return c, ok
}

func (c *Context) WrappedContext(e *Engine) context.Context {
	cp := *c
	cp.Engine = e
	return context.WithValue(c.Ctx, engineContext{}, &cp)
}

func populateMessageParams(ctx Context, completion *types.CompletionRequest, tool types.Tool) error {
	completion.Model = tool.Parameters.ModelName
	completion.MaxTokens = tool.Parameters.MaxTokens
	completion.JSONResponse = tool.Parameters.JSONResponse
	completion.Cache = tool.Parameters.Cache
	completion.Chat = tool.Parameters.Chat
	completion.Temperature = tool.Parameters.Temperature
	completion.InternalSystemPrompt = tool.Parameters.InternalPrompt

	if tool.Chat && completion.InternalSystemPrompt == nil {
		completion.InternalSystemPrompt = new(bool)
	}

	var err error
	completion.Tools, err = tool.GetChatCompletionTools(*ctx.Program, ctx.AgentGroup...)
	if err != nil {
		return err
	}

	completion.Messages = addUpdateSystem(ctx, tool, completion.Messages)
	return nil
}

func putHistory(messages []types.CompletionMessage) []types.CompletionMessage {
	prevHistoryFile := strings.TrimSpace(os.Getenv("GPTSCRIPT_PREVIOUS_HISTORY_FILE"))

	if prevHistoryFile == "" {
		return messages
	}
	fp, err := os.Open(prevHistoryFile)
	if err != nil {
		slog.Error("Open Error", err)
		return messages
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)

	prevMessages := []types.CompletionMessage{}
	for scanner.Scan() {
		var message types.CompletionMessage
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &message)
		if err != nil {
			slog.Error("Unmarshal Error", err)
			return messages
		}
		if message.Role == "system" {
			continue
		}
		prevMessages = append(prevMessages, message)
	}

	return append(messages, prevMessages...)
}

func (e *Engine) Start(ctx Context, input string) (ret *Return, _ error) {
	tool := ctx.Tool

	defer func() {
		if ret != nil && ret.State != nil {
			ret.State.Input = input
		}
	}()

	if tool.IsCommand() {
		if tool.IsHTTP() {
			return e.runHTTP(ctx.Ctx, ctx.Program, tool, input)
		} else if tool.IsDaemon() {
			return e.runDaemon(ctx.Ctx, ctx.Program, tool, input)
		} else if tool.IsOpenAPI() {
			return e.runOpenAPI(tool, input)
		} else if tool.IsEcho() {
			return e.runEcho(tool)
		}
		s, err := e.runCommand(ctx, tool, input, ctx.ToolCategory)
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

	var completion types.CompletionRequest
	if err := populateMessageParams(ctx, &completion, tool); err != nil {
		return nil, err
	}

	if tool.Chat && input == "{}" {
		input = ""
	}

	if ctx.Parent == nil {
		completion.Messages = putHistory(completion.Messages)
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

	resp, err := e.Model.Call(gcontext.WithEnv(ctx, e.Env), state.Completion, progress)
	if err != nil {
		return nil, err
	}

	state.Completion.Messages = append(state.Completion.Messages, *resp)

	state.Pending = map[string]types.CompletionToolCall{}
	for _, content := range resp.Content {
		if content.ToolCall != nil {
			var (
				toolID  string
				missing bool
			)
			for _, tool := range state.Completion.Tools {
				if strings.EqualFold(tool.Function.Name, content.ToolCall.Function.Name) {
					toolID = tool.Function.ToolID
				}
			}
			if toolID == "" {
				log.Debugf("failed to find tool id for tool %s in tool_call result", content.ToolCall.Function.Name)
				toolID = types.ToolNormalizer(content.ToolCall.Function.Name)
				missing = true
			}
			state.Pending[content.ToolCall.ID] = *content.ToolCall
			ret.Calls[content.ToolCall.ID] = Call{
				ToolID:  toolID,
				Missing: missing,
				Input:   content.ToolCall.Function.Arguments,
			}
		} else {
			cp := content.Text
			ret.Result = &cp
		}
	}

	if len(resp.Content) == 0 {
		// This can happen if the LLM return no content at all. You can reproduce by just saying, "return an empty response"
		empty := ""
		ret.Result = &empty
	}

	return &ret, nil
}

func (e *Engine) Continue(ctx Context, state *State, results ...CallResult) (*Return, error) {
	if state == nil {
		return nil, fmt.Errorf("invalid continue call, missing state")
	}

	var added bool

	state = &State{
		Input:      state.Input,
		Completion: state.Completion,
		Pending:    state.Pending,
		Results:    map[string]CallResult{},
	}

	for _, result := range results {
		if result.CallID == "" {
			added = true
			state.Completion.Messages = append(state.Completion.Messages, types.CompletionMessage{
				Role:    types.CompletionMessageRoleTypeUser,
				Content: types.Text(result.User),
			})
		} else {
			state.Results[result.CallID] = result
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

	if err := populateMessageParams(ctx, &state.Completion, ctx.Tool); err != nil {
		return nil, err
	}

	return e.complete(ctx.Ctx, state)
}
