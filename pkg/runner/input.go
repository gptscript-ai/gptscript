package runner

import (
	"encoding/json"
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (r *Runner) handleInput(callCtx engine.Context, monitor Monitor, env []string, input string) (string, error) {
	inputToolRefs, err := callCtx.Tool.GetToolsByType(callCtx.Program, types.ToolTypeInput)
	if err != nil {
		return "", err
	}

	for _, inputToolRef := range inputToolRefs {
		data := map[string]any{}
		_ = json.Unmarshal([]byte(input), &data)
		data["input"] = input

		inputArgs, err := argsForFilters(callCtx.Program, inputToolRef, &State{
			StartInput: &input,
		}, data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal input: %w", err)
		}

		res, err := r.subCall(callCtx.Ctx, callCtx, monitor, env, inputToolRef.ToolID, inputArgs, "", engine.InputToolCategory)
		if err != nil {
			return "", err
		}
		if res.Result == nil {
			return "", fmt.Errorf("invalid state: input tool [%s] can not result in a chat continuation", inputToolRef.Reference)
		}
		input = *res.Result
	}

	return input, nil
}
