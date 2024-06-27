package types

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/getkin/kin-openapi/openapi3"
)

type CompletionRequest struct {
	Model                string              `json:"model,omitempty"`
	InternalSystemPrompt *bool               `json:"internalSystemPrompt,omitempty"`
	Tools                []CompletionTool    `json:"tools,omitempty"`
	Messages             []CompletionMessage `json:"messages,omitempty"`
	MaxTokens            int                 `json:"maxTokens,omitempty"`
	Chat                 bool                `json:"chat,omitempty"`
	Temperature          *float32            `json:"temperature,omitempty"`
	JSONResponse         bool                `json:"jsonResponse,omitempty"`
	Cache                *bool               `json:"cache,omitempty"`
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
	Content []ContentPart             `json:"content,omitempty"`
	// ToolCall should be set for only messages of type "tool" and Content[0].Text should be set as the
	// result of the call describe by this field
	ToolCall *CompletionToolCall `json:"toolCall,omitempty"`
	Usage    Usage               `json:"usage,omitempty"`
}

func (c CompletionMessage) ChatText() string {
	var buf strings.Builder
	for _, part := range c.Content {
		if part.Text == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(part.Text)
	}
	return buf.String()
}

type Usage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

type CompletionStatus struct {
	CompletionID    string
	Request         any
	Response        any
	Usage           Usage
	Cached          bool
	Chunks          any
	PartialResponse *CompletionMessage
}

func (c CompletionMessage) IsToolCall() bool {
	for _, content := range c.Content {
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

func (c CompletionMessage) String() string {
	buf := strings.Builder{}
	for i, content := range c.Content {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(content.Text)
		if content.ToolCall != nil {
			buf.WriteString(fmt.Sprintf("<tool call> %s -> %s", color.GreenString(content.ToolCall.Function.Name), content.ToolCall.Function.Arguments))
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
