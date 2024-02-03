package types

import (
	"context"
	"fmt"
	"strings"
)

type ToolSet map[string]Tool

type Program struct {
	EntryToolID string  `json:"entryToolId,omitempty"`
	ToolSet     ToolSet `json:"toolSet,omitempty"`
}

type BuiltinFunc func(ctx context.Context, env []string, input string) (string, error)

type Tool struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Arguments    *JSONSchema       `json:"arguments,omitempty"`
	Instructions string            `json:"instructions,omitempty"`
	Tools        []string          `json:"tools,omitempty"`
	ToolMapping  map[string]string `json:"toolMapping,omitempty"`
	BuiltinFunc  BuiltinFunc       `json:"-"`

	Vision       bool   `json:"vision,omitempty"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
	ModelName    string `json:"modelName,omitempty"`
	JSONResponse bool   `json:"jsonResponse,omitempty"`
	Cache        *bool  `json:"cache,omitempty"`

	Source ToolSource `json:"source,omitempty"`
}

type ToolSource struct {
	File   string `json:"file,omitempty"`
	LineNo int    `json:"lineNo,omitempty"`
}

func (t ToolSource) String() string {
	return fmt.Sprintf("%s:%d", t.File, t.LineNo)
}

func (t Tool) IsCommand() bool {
	return strings.HasPrefix(t.Instructions, "#!")
}

func FirstSet[T comparable](in ...T) (result T) {
	for _, i := range in {
		if i != result {
			return i
		}
	}
	return
}
