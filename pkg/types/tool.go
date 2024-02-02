package types

import (
	"strings"
)

type ToolSet map[string]Tool

type Tool struct {
	Name         string      `json:"name,omitempty"`
	Description  string      `json:"description,omitempty"`
	Arguments    *JSONSchema `json:"arguments,omitempty"`
	Instructions string      `json:"instructions,omitempty"`
	Tools        []string    `json:"tools,omitempty"`

	Vision       bool   `json:"vision,omitempty"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
	ModelName    string `json:"modelName,omitempty"`
	JSONResponse bool   `json:"jsonResponse,omitempty"`
	Cache        *bool  `json:"cache,omitempty"`

	ToolSet ToolSet    `json:"toolSet,omitempty"`
	Source  ToolSource `json:"source,omitempty"`
}

type ToolSource struct {
	File   string `json:"file,omitempty"`
	LineNo int    `json:"lineNo,omitempty"`
}

func (t Tool) IsCommand() bool {
	return strings.HasPrefix(t.Instructions, "#!")
}
