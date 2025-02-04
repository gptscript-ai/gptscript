package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runCall(ctx Context, tool types.Tool, input string) (*Return, error) {
	interpreter, body, _ := strings.Cut(tool.Instructions, "\n")

	fields := strings.Fields(interpreter)
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid tool call, no target tool found in %s", tool.Instructions)
	}
	toolRef := strings.Join(fields[1:], " ")

	toolName, args := types.SplitArg(toolRef)

	toolNameParts := strings.Fields(toolName)

	toolName = toolNameParts[0]
	toolNameArgs := toolNameParts[1:]

	targetTools, ok := tool.ToolMapping[toolName]
	if !ok || len(targetTools) == 0 {
		return nil, fmt.Errorf("target tool %s not found, must reference in `tools:` fields", toolName)
	}

	ref := types.ToolReference{
		Reference: toolName,
		Arg:       args,
		ToolID:    targetTools[0].ToolID,
	}

	newInput, err := types.GetToolRefInput(ctx.Program, ref, input)
	if err != nil {
		return nil, err
	}

	newInput, err = mergeInputs(input, newInput)
	if err != nil {
		return nil, fmt.Errorf("failed to merge inputs: %w", err)
	}

	newInput, err = mergeInputs(newInput, toString(map[string]string{
		"TOOL_CALL_ARGS": strings.Join(toolNameArgs, " "),
		"TOOL_CALL_BODY": body,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to merge inputs for tool calls: %w", err)
	}

	newCtx := ctx
	newCtx.Tool = ctx.Program.ToolSet[ref.ToolID]

	return e.Start(newCtx, newInput)
}

func toString(data map[string]string) string {
	out, err := json.Marshal(data)
	if err != nil {
		// this will never happen
		panic(err)
	}
	return string(out)
}

func mergeInputs(base, overlay string) (string, error) {
	baseMap := map[string]interface{}{}
	overlayMap := map[string]interface{}{}

	if overlay == "" || overlay == "{}" {
		return base, nil
	}

	if base != "" {
		if err := json.Unmarshal([]byte(base), &baseMap); err != nil {
			return "", fmt.Errorf("failed to unmarshal base input: %w", err)
		}
	}

	if err := json.Unmarshal([]byte(overlay), &overlayMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal overlay input: %w", err)
	}

	for k, v := range overlayMap {
		baseMap[k] = v
	}

	out, err := json.Marshal(baseMap)
	return string(out), err
}
