package types

import (
	"context"
	"fmt"
	"sort"
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

	Vision       bool     `json:"vision,omitempty"`
	MaxTokens    int      `json:"maxTokens,omitempty"`
	ModelName    string   `json:"modelName,omitempty"`
	JSONResponse bool     `json:"jsonResponse,omitempty"`
	Temperature  *float32 `json:"temperature,omitempty"`
	Cache        *bool    `json:"cache,omitempty"`

	Source ToolSource `json:"source,omitempty"`
}

func (t Tool) String() string {
	buf := &strings.Builder{}
	if t.Name != "" {
		_, _ = fmt.Fprintf(buf, "Name: %s\n", t.Name)
	}
	if t.Description != "" {
		_, _ = fmt.Fprintf(buf, "Description: %s\n", t.Name)
	}
	if len(t.Tools) != 0 {
		_, _ = fmt.Fprintf(buf, "Tools: %s\n", strings.Join(t.Tools, ", "))
	}
	if t.Vision {
		_, _ = fmt.Fprintln(buf, "Vision: true")
	}
	if t.MaxTokens != 0 {
		_, _ = fmt.Fprintf(buf, "Max Tokens: %d\n", t.MaxTokens)
	}
	if t.ModelName != "" {
		_, _ = fmt.Fprintf(buf, "Model Name: %s\n", t.ModelName)
	}
	if t.JSONResponse {
		_, _ = fmt.Fprintln(buf, "JSON Response: true")
	}
	if t.Cache != nil && !*t.Cache {
		_, _ = fmt.Fprintln(buf, "Cache: false")
	}
	if t.Temperature != nil {
		_, _ = fmt.Fprintf(buf, "Temperature: %f", *t.Temperature)
	}
	if t.Arguments != nil {
		var keys []string
		for k := range t.Arguments.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			prop := t.Arguments.Properties[key]
			_, _ = fmt.Fprintf(buf, "Args: %s: %s\n", key, prop.Description)
		}
	}
	if t.Instructions != "" && t.BuiltinFunc == nil {
		_, _ = fmt.Fprintln(buf)
		_, _ = fmt.Fprintln(buf, t.Instructions)
	}

	return buf.String()
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
