package types

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const (
	DaemonPrefix  = "#!sys.daemon"
	CommandPrefix = "#!"
)

type ToolSet map[string]Tool

type Program struct {
	Name        string            `json:"name,omitempty"`
	EntryToolID string            `json:"entryToolId,omitempty"`
	ToolSet     ToolSet           `json:"toolSet,omitempty"`
	Exports     map[string]string `json:"exports,omitempty"`
}

type BuiltinFunc func(ctx context.Context, env []string, input string) (string, error)

type Parameters struct {
	Name           string      `json:"name,omitempty"`
	Description    string      `json:"description,omitempty"`
	MaxTokens      int         `json:"maxTokens,omitempty"`
	ModelName      string      `json:"modelName,omitempty"`
	JSONResponse   bool        `json:"jsonResponse,omitempty"`
	Temperature    *float32    `json:"temperature,omitempty"`
	Cache          *bool       `json:"cache,omitempty"`
	InternalPrompt *bool       `json:"internalPrompt"`
	Arguments      *JSONSchema `json:"arguments,omitempty"`
	Tools          []string    `json:"tools,omitempty"`
}

type Tool struct {
	Parameters   `json:",inline"`
	Instructions string `json:"instructions,omitempty"`

	ID          string            `json:"id,omitempty"`
	ToolMapping map[string]string `json:"toolMapping,omitempty"`
	LocalTools  map[string]string `json:"localTools,omitempty"`
	BuiltinFunc BuiltinFunc       `json:"-"`
	Source      ToolSource        `json:"source,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`
}

func (t Tool) String() string {
	buf := &strings.Builder{}
	if t.Parameters.Name != "" {
		_, _ = fmt.Fprintf(buf, "Name: %s\n", t.Parameters.Name)
	}
	if t.Parameters.Description != "" {
		_, _ = fmt.Fprintf(buf, "Description: %s\n", t.Parameters.Description)
	}
	if len(t.Parameters.Tools) != 0 {
		_, _ = fmt.Fprintf(buf, "Tools: %s\n", strings.Join(t.Parameters.Tools, ", "))
	}
	if t.Parameters.MaxTokens != 0 {
		_, _ = fmt.Fprintf(buf, "Max Tokens: %d\n", t.Parameters.MaxTokens)
	}
	if t.Parameters.ModelName != "" {
		_, _ = fmt.Fprintf(buf, "Model Name: %s\n", t.Parameters.ModelName)
	}
	if t.Parameters.JSONResponse {
		_, _ = fmt.Fprintln(buf, "JSON Response: true")
	}
	if t.Parameters.Cache != nil && !*t.Parameters.Cache {
		_, _ = fmt.Fprintln(buf, "Cache: false")
	}
	if t.Parameters.Temperature != nil {
		_, _ = fmt.Fprintf(buf, "Temperature: %f", *t.Parameters.Temperature)
	}
	if t.Parameters.Arguments != nil {
		var keys []string
		for k := range t.Parameters.Arguments.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			prop := t.Parameters.Arguments.Properties[key]
			_, _ = fmt.Fprintf(buf, "Args: %s: %s\n", key, prop.Description)
		}
	}
	if t.Parameters.InternalPrompt != nil {
		_, _ = fmt.Fprintf(buf, "Internal Prompt: %v\n", *t.Parameters.InternalPrompt)
	}
	if t.Instructions != "" && t.BuiltinFunc == nil {
		_, _ = fmt.Fprintln(buf)
		_, _ = fmt.Fprintln(buf, t.Instructions)
	}

	return buf.String()
}

type ToolSource struct {
	Location string `json:"location,omitempty"`
	LineNo   int    `json:"lineNo,omitempty"`
}

func (t ToolSource) String() string {
	return fmt.Sprintf("%s:%d", t.Location, t.LineNo)
}

func (t Tool) IsCommand() bool {
	return strings.HasPrefix(t.Instructions, CommandPrefix)
}

func (t Tool) IsDaemon() bool {
	return strings.HasPrefix(t.Instructions, DaemonPrefix)
}

func (t Tool) IsHTTP() bool {
	return strings.HasPrefix(t.Instructions, "#!http://") ||
		strings.HasPrefix(t.Instructions, "#!https://")
}

func FirstSet[T comparable](in ...T) (result T) {
	for _, i := range in {
		if i != result {
			return i
		}
	}
	return
}
