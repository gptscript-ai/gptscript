package types

import (
	"fmt"
	"strings"
)

const (
	CompletionToolTypeFunction CompletionToolType = "function"
)

type CompletionToolType string

type CompletionRequest struct {
	Model        string
	Vision       bool
	Tools        []CompletionTool
	Messages     []CompletionMessage
	MaxToken     int
	JSONResponse bool
	Cache        *bool
}

type CompletionTool struct {
	Type     CompletionToolType           `json:"type"`
	Function CompletionFunctionDefinition `json:"function,omitempty"`
}

type CompletionFunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Domain      string      `json:"domain,omitempty"`
	Parameters  *JSONSchema `json:"parameters"`
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
	Role     CompletionMessageRoleType `json:"role,omitempty"`
	Content  []ContentPart             `json:"content,omitempty" column:"name=Message,jsonpath=.spec.content"`
	ToolCall *CompletionToolCall       `json:"toolCall,omitempty"`
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
	if !in.HasContent() {
		return ""
	}
	buf := strings.Builder{}
	if in.Role == CompletionMessageRoleTypeUser {
		buf.WriteString("input -> ")
	} else if in.Role == CompletionMessageRoleTypeTool && in.ToolCall != nil {
		buf.WriteString(fmt.Sprintf("tool return %s -> ", in.ToolCall.Function.Name))
	}
	for i, content := range in.Content {
		if i > 0 {
			buf.WriteString("\n -> ")
		}
		buf.WriteString(content.Text)
		if content.ToolCall != nil {
			buf.WriteString(fmt.Sprintf("tool call %s -> %s", content.ToolCall.Function.Name, content.ToolCall.Function.Arguments))
		}
		if content.Image != nil {
			buf.WriteString("image: ")
			if content.Image.URL != "" {
				buf.WriteString(content.Image.URL)
			}
			if len(content.Image.Base64) > 50 {
				buf.WriteString(content.Image.Base64[:50] + "...")
			} else {
				buf.WriteString(content.Image.Base64)
			}
		}
	}
	return buf.String()
}

type ContentPart struct {
	Text     string              `json:"text,omitempty"`
	ToolCall *CompletionToolCall `json:"toolCall,omitempty"`
	Image    *ImageURL           `json:"image,omitempty"`
}

func (in ContentPart) HasContent() bool {
	return in.Text != "" || in.ToolCall != nil || in.Image != nil
}

func (in CompletionMessage) HasContent() bool {
	return len(in.Content) > 0 && in.Content[0].HasContent()
}

type ImageURLDetail string

const (
	ImageURLDetailHigh ImageURLDetail = "high"
	ImageURLDetailLow  ImageURLDetail = "low"
	ImageURLDetailAuto ImageURLDetail = "auto"
)

type ImageURL struct {
	Base64      string         `json:"base64,omitempty"`
	ContentType string         `json:"contentType,omitempty"`
	URL         string         `json:"url,omitempty"`
	Detail      ImageURLDetail `json:"detail,omitempty"`
}

type CompletionToolCall struct {
	Index    *int                   `json:"index,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Type     CompletionToolType     `json:"type,omitempty"`
	Function CompletionFunctionCall `json:"function,omitempty"`
}

type CompletionFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
