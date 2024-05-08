package types

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/getkin/kin-openapi/openapi3"
)

type CompletionRequest struct {
	Model                string
	InternalSystemPrompt *bool
	Tools                []CompletionTool
	Messages             []CompletionMessage
	MaxTokens            int
	Temperature          *float32
	JSONResponse         bool
	Grammar              string
	Cache                *bool
}

func (r *CompletionRequest) GetCache() bool {
	if r.Cache == nil {
		return true
	}
	return *r.Cache
}

type CompletionTool struct {
	Function CompletionFunctionDefinition `json:"function,omitempty"`
}

type CompletionFunctionDefinition struct {
	ToolID      string           `json:"toolID,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Parameters  *openapi3.Schema `json:"parameters"`
}

// Chat message role defined by the OpenAI API.
const (
	CompletionMessageRoleTypeUser      = CompletionMessageRoleType("user")
	CompletionMessageRoleTypeSystem    = CompletionMessageRoleType("system")
	CompletionMessageRoleTypeAssistant = CompletionMessageRoleType("assistant")
	CompletionMessageRoleTypeTool      = CompletionMessageRoleType("tool")
)

type CompletionMessageRoleType string

type CompletionMessage struct {
	Role    CompletionMessageRoleType `json:"role,omitempty"`
	Content []ContentPart             `json:"content,omitempty" column:"name=Message,jsonpath=.spec.content"`
	// ToolCall should be set for only messages of type "tool" and Content[0].Text should be set as the
	// result of the call describe by this field
	ToolCall *CompletionToolCall `json:"toolCall,omitempty"`
}

type CompletionStatus struct {
	CompletionID    string
	Request         any
	Response        any
	Cached          bool
	Chunks          any
	PartialResponse *CompletionMessage
}

func (in CompletionMessage) IsToolCall() bool {
	for _, content := range in.Content {
		if content.ToolCall != nil {
			return true
		}
	}
	return false
}

func Text(text string) []ContentPart {
	return []ContentPart{
		{
			Text: text,
		},
	}
}

func (in CompletionMessage) String() string {
	buf := strings.Builder{}
	for i, content := range in.Content {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(content.Text)
		if content.ToolCall != nil {
			buf.WriteString(fmt.Sprintf("tool call %s -> %s", color.GreenString(content.ToolCall.Function.Name), content.ToolCall.Function.Arguments))
		}
	}
	return buf.String()
}

type ContentPart struct {
	Text     string              `json:"text,omitempty"`
	ToolCall *CompletionToolCall `json:"toolCall,omitempty"`
}

type CompletionToolCall struct {
	Index    *int                   `json:"index,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Function CompletionFunctionCall `json:"function,omitempty"`
}

type CompletionFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
