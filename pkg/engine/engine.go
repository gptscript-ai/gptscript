package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/types"
	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/google/shlex"
)

// InternalSystemPrompt is added to all threads. Changing this is very dangerous as it has a
// terrible global effect and changes the behavior of all scripts.
var InternalSystemPrompt = `
You are task oriented system.
You receive input from a user, process the input from the given instructions, and then output the result.
Your objective is to provide consist and correct results.
You do not need to explain the steps taken, only provide the result to the given instructions.
You are referred to as a tool.
`

var completionID int64

func init() {
	if p := os.Getenv("GPTSCRIPT_INTERNAL_SYSTEM_PROMPT"); p != "" {
		InternalSystemPrompt = p
	}
}

type ErrToolNotFound struct {
	ToolName string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

type Engine struct {
	Client   *openai.Client
	Env      []string
	Progress chan<- openai.Status
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
	ToolName string `json:"toolName,omitempty"`
	Input    string `json:"input,omitempty"`
}

type CallResult struct {
	ID     string `json:"id,omitempty"`
	Result string `json:"result,omitempty"`
}

type Context struct {
	ID      string
	Ctx     context.Context
	Parent  *Context
	Program *types.Program
	Tool    types.Tool
}

func (c *Context) ParentID() string {
	if c.Parent == nil {
		return ""
	}
	return c.Parent.ID
}

func (c *Context) UnmarshalJSON(data []byte) error {
	panic("this data struct is circular by design and can not be read from json")
}

func (c *Context) MarshalJSON() ([]byte, error) {
	var parentID string
	if c.Parent != nil {
		parentID = c.Parent.ID
	}
	return json.Marshal(map[string]any{
		"id":       c.ID,
		"parentID": parentID,
		"tool":     c.Tool,
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

func (c *Context) SubCall(ctx context.Context, toolName, callID string) (Context, error) {
	tool, err := c.getTool(toolName)
	if err != nil {
		return Context{}, err
	}
	return Context{
		ID:      callID,
		Ctx:     ctx,
		Parent:  c,
		Program: c.Program,
		Tool:    tool,
	}, nil
}

func (c *Context) getTool(name string) (types.Tool, error) {
	toolID, ok := c.Tool.ToolMapping[name]
	if !ok {
		return types.Tool{}, &ErrToolNotFound{
			ToolName: name,
		}
	}
	tool, ok := c.Program.ToolSet[toolID]
	if !ok {
		return types.Tool{}, &ErrToolNotFound{
			ToolName: name,
		}
	}
	return tool, nil
}

func (e *Engine) runCommand(ctx context.Context, tool types.Tool, input string) (cmdOut string, cmdErr error) {
	id := fmt.Sprint(atomic.AddInt64(&completionID, 1))

	defer func() {
		e.Progress <- openai.Status{
			CompletionID: id,
			Response: map[string]any{
				"output": cmdOut,
				"err":    cmdErr,
			},
		}
	}()

	if tool.BuiltinFunc != nil {
		e.Progress <- openai.Status{
			CompletionID: id,
			Request: map[string]any{
				"command": []string{tool.ID},
				"input":   input,
			},
		}
		return tool.BuiltinFunc(ctx, e.Env, input)
	}

	env := e.Env[:]
	data := map[string]any{}

	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()

	envMap := map[string]string{}
	if err := json.Unmarshal([]byte(input), &data); err == nil {
		for k, v := range data {
			envName := strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
			switch val := v.(type) {
			case string:
				envMap[envName] = val
				env = append(env, envName+"="+val)
				envMap[k] = val
				env = append(env, k+"="+val)
			case json.Number:
				envMap[envName] = string(val)
				env = append(env, envName+"="+string(val))
				envMap[k] = string(val)
				env = append(env, k+"="+string(val))
			case bool:
				envMap[envName] = fmt.Sprint(val)
				env = append(env, envName+"="+fmt.Sprint(val))
				envMap[k] = fmt.Sprint(val)
				env = append(env, k+"="+fmt.Sprint(val))
			default:
				data, err := json.Marshal(val)
				if err == nil {
					envMap[envName] = string(data)
					env = append(env, envName+"="+string(data))
					envMap[k] = string(data)
					env = append(env, k+"="+string(data))
				}
			}
		}
	}

	interpreter, rest, _ := strings.Cut(tool.Instructions, "\n")
	f, err := os.CreateTemp("", version.ProgramName)
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(rest))
	_ = f.Close()
	if err != nil {
		return "", err
	}
	interpreter = strings.TrimSpace(interpreter)[2:]

	interpreter = os.Expand(interpreter, func(s string) string {
		return envMap[s]
	})

	args, err := shlex.Split(interpreter)
	if err != nil {
		return "", err
	}

	e.Progress <- openai.Status{
		CompletionID: id,
		Request: map[string]any{
			"command": args,
			"input":   input,
		},
	}

	output := &bytes.Buffer{}

	cmd := exec.Command(args[0], append(args[1:], f.Name())...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	cmd.Stdout = output

	if err := cmd.Run(); err != nil {
		_, _ = os.Stderr.Write(output.Bytes())
		log.Errorf("failed to run tool [%s] cmd [%s]: %v", tool.Name, interpreter, err)
		return "", err
	}

	return output.String(), nil
}

func (e *Engine) Start(ctx Context, input string) (*Return, error) {
	tool := ctx.Tool

	if tool.IsCommand() {
		s, err := e.runCommand(ctx.Ctx, tool, input)
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
		MaxToken:     tool.MaxTokens,
		JSONResponse: tool.JSONResponse,
		Cache:        tool.Cache,
		Temperature:  tool.Temperature,
	}

	if InternalSystemPrompt != "" {
		completion.Messages = append(completion.Messages, types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeSystem,
			Content: types.Text(InternalSystemPrompt),
		})
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
		progress = make(chan openai.Status)
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
	state = &State{
		Completion: state.Completion,
		Pending:    state.Pending,
		Results:    map[string]CallResult{},
	}

	for _, result := range results {
		state.Results[result.ID] = result
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
				ToolName: pending.Function.Name,
				Input:    pending.Function.Arguments,
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
			return nil, fmt.Errorf("missing tool call pennding for id %s, most likely a %s BUG",
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
