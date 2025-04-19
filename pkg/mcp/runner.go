package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/mark3labs/mcp-go/mcp"
)

func (l *Local) Run(ctx context.Context, _ chan<- types.CompletionStatus, tool types.Tool, input string) (string, error) {
	fields := strings.Fields(tool.Instructions)
	if len(fields) < 3 {
		return "", fmt.Errorf("invalid mcp call, invalid number of fields in %s", tool.Instructions)
	}

	id := fields[1]
	toolName := fields[2]
	arguments := map[string]any{}

	if input != "" {
		if err := json.Unmarshal([]byte(input), &arguments); err != nil {
			return "", fmt.Errorf("failed to unmarshal input: %w", err)
		}
	}

	l.lock.Lock()
	session, ok := l.sessions[id]
	l.lock.Unlock()
	if !ok {
		return "", fmt.Errorf("session not found for MCP server %s", id)
	}

	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = arguments

	result, err := session.Client.CallTool(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to call tool %s: %w", toolName, err)
	}

	str, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(str), nil
}
