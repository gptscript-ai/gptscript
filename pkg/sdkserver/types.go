package sdkserver

import (
	"maps"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	gserver "github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type runState string

const (
	Creating runState = "creating"
	Running  runState = "running"
	Continue runState = "continue"
	Finished runState = "finished"
	Error    runState = "error"

	CallConfirm runner.EventType = "callConfirm"
)

type toolOrFileRequest struct {
	cache.Options `json:",inline"`
	types.ToolDef `json:",inline"`
	content       `json:",inline"`
	file          `json:",inline"`

	SubTool           string   `json:"subTool"`
	Input             string   `json:"input"`
	ChatState         string   `json:"chatState"`
	Workspace         string   `json:"workspace"`
	Env               []string `json:"env"`
	CredentialContext string   `json:"credentialContext"`
	Confirm           bool     `json:"confirm"`
}

type content struct {
	Content string `json:"content"`
}

func (c *content) String() string {
	return c.Content
}

type file struct {
	File string `json:"file"`
}

func (f *file) String() string {
	return f.File
}

type parseRequest struct {
	parser.Options `json:",inline"`
	content        `json:",inline"`

	File string `json:"file"`
}

type modelsRequest struct {
	Providers []string `json:"providers"`
}

type runInfo struct {
	Calls     map[string]call `json:"-"`
	ID        string          `json:"id"`
	Program   types.Program   `json:"program"`
	Input     string          `json:"input"`
	Output    string          `json:"output"`
	Error     string          `json:"error"`
	Start     time.Time       `json:"start"`
	End       time.Time       `json:"end"`
	State     runState        `json:"state"`
	ChatState any             `json:"chatState"`
}

func newRun(id string) *runInfo {
	return &runInfo{
		ID:    id,
		State: Creating,
		Calls: make(map[string]call),
	}
}

type runEvent struct {
	runInfo `json:",inline"`

	Type runner.EventType `json:"type"`
}

func (r *runInfo) process(event gserver.Event) map[string]any {
	switch event.Type {
	case runner.EventTypeRunStart:
		r.Start = event.Time
		r.Program = *event.Program
		r.State = Running
	case runner.EventTypeRunFinish:
		r.End = event.Time
		r.Output = event.Output
		r.Error = event.Err
		if r.Error != "" {
			r.State = Error
		} else {
			r.State = Finished
		}
	}

	if event.CallContext == nil || event.CallContext.ID == "" {
		return map[string]any{"run": runEvent{
			runInfo: *r,
			Type:    event.Type,
		}}
	}

	call := r.Calls[event.CallContext.ID]
	call.CallContext = *event.CallContext
	call.Type = event.Type

	switch event.Type {
	case runner.EventTypeCallStart:
		call.Start = event.Time
		call.Input = event.Content

	case runner.EventTypeCallSubCalls:
		call.setSubCalls(event.ToolSubCalls)

	case runner.EventTypeCallProgress:
		call.setOutput(event.Content)

	case runner.EventTypeCallFinish:
		call.End = event.Time
		call.setOutput(event.Content)

	case runner.EventTypeChat:
		if event.ChatRequest != nil {
			call.LLMRequest = event.ChatRequest
		}
		if event.ChatResponse != nil {
			call.LLMResponse = event.ChatResponse
		}
	}

	r.Calls[event.CallContext.ID] = call
	return map[string]any{"call": call}
}

func (r *runInfo) processStdout(cs runner.ChatResponse) {
	if cs.Done {
		r.State = Finished
	} else {
		r.State = Continue
	}

	r.ChatState = cs.State
}

type call struct {
	engine.CallContext `json:",inline"`

	Type        runner.EventType `json:"type"`
	Start       time.Time        `json:"start"`
	End         time.Time        `json:"end"`
	Input       string           `json:"input"`
	Output      []output         `json:"output"`
	Usage       types.Usage      `json:"usage"`
	LLMRequest  any              `json:"llmRequest"`
	LLMResponse any              `json:"llmResponse"`
}

func (c *call) setSubCalls(subCalls map[string]engine.Call) {
	c.Output = append(c.Output, output{
		SubCalls: maps.Clone(subCalls),
	})
}

func (c *call) setOutput(o string) {
	if len(c.Output) == 0 || len(c.Output[len(c.Output)-1].SubCalls) > 0 {
		c.Output = append(c.Output, output{
			Content: o,
		})
	} else {
		c.Output[len(c.Output)-1].Content = o
	}
}

type output struct {
	Content  string                 `json:"content"`
	SubCalls map[string]engine.Call `json:"subCalls"`
}
