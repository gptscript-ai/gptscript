package sdkserver

import (
	"maps"
	"strings"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/openai"
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
	Prompt      runner.EventType = "prompt"
)

type toolDefs []types.ToolDef

func (t toolDefs) String() string {
	s := new(strings.Builder)
	for i, tool := range t {
		s.WriteString(tool.String())
		if i != len(t)-1 {
			s.WriteString("\n\n---\n\n")
		}
	}

	return s.String()
}

type (
	cacheOptions  cache.Options
	openAIOptions openai.Options
)

type toolOrFileRequest struct {
	content       `json:",inline"`
	file          `json:",inline"`
	cacheOptions  `json:",inline"`
	openAIOptions `json:",inline"`

	ToolDefs            toolDefs `json:"toolDefs,inline"`
	SubTool             string   `json:"subTool"`
	Input               string   `json:"input"`
	ChatState           string   `json:"chatState"`
	Workspace           string   `json:"workspace"`
	Env                 []string `json:"env"`
	CredentialContext   string   `json:"credentialContext"`
	CredentialOverrides []string `json:"credentialOverrides"`
	Confirm             bool     `json:"confirm"`
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
	Type    runner.EventType `json:"type"`
}

func (r *runInfo) process(e event) map[string]any {
	switch e.Type {
	case Prompt:
		return map[string]any{"prompt": prompt{
			Prompt: e.Prompt,
			ID:     e.RunID,
			Type:   e.Type,
			Time:   e.Time,
		}}
	case runner.EventTypeRunStart:
		r.Start = e.Time
		r.Program = *e.Program
		r.State = Running
	case runner.EventTypeRunFinish:
		r.End = e.Time
		r.Output = e.Output
		r.Error = e.Err
		if r.Error != "" {
			r.State = Error
		} else {
			r.State = Finished
		}
	}

	if e.CallContext == nil || e.CallContext.ID == "" {
		return map[string]any{"run": runEvent{
			runInfo: *r,
			Type:    e.Type,
		}}
	}

	call := r.Calls[e.CallContext.ID]
	call.CallContext = *e.CallContext
	call.Type = e.Type

	switch e.Type {
	case runner.EventTypeCallStart:
		call.Start = e.Time
		call.Input = e.Content

	case runner.EventTypeCallSubCalls:
		call.setSubCalls(e.ToolSubCalls)

	case runner.EventTypeCallProgress:
		call.setOutput(e.Content)

	case runner.EventTypeCallFinish:
		call.End = e.Time
		call.setOutput(e.Content)

	case runner.EventTypeChat:
		if e.ChatRequest != nil {
			call.LLMRequest = e.ChatRequest
		}
		if e.ChatResponse != nil {
			call.LLMResponse = e.ChatResponse
		}
	}

	r.Calls[e.CallContext.ID] = call
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

type event struct {
	gserver.Event `json:",inline"`
	types.Prompt  `json:",inline"`
}

type prompt struct {
	types.Prompt `json:",inline"`
	ID           string           `json:"id,omitempty"`
	Type         runner.EventType `json:"type,omitempty"`
	Time         time.Time        `json:"time,omitempty"`
}
