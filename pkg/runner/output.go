package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func argsForFilters(prg *types.Program, tool types.ToolReference, startState *State, filterDefinedInput map[string]any) (string, error) {
	startInput := ""
	if startState.ResumeInput != nil {
		startInput = *startState.ResumeInput
	} else if startState.StartInput != nil {
		startInput = *startState.StartInput
	}

	parsedArgs, err := types.GetToolRefInput(prg, tool, startInput)
	if err != nil {
		return "", err
	}

	argData := map[string]any{}
	if strings.HasPrefix(parsedArgs, "{") {
		if err := json.Unmarshal([]byte(parsedArgs), &argData); err != nil {
			return "", fmt.Errorf("failed to unmarshal parsedArgs for filter: %w", err)
		}
	} else if _, hasInput := filterDefinedInput["input"]; parsedArgs != "" && !hasInput {
		argData["input"] = parsedArgs
	}

	resultData := map[string]any{}
	maps.Copy(resultData, filterDefinedInput)
	maps.Copy(resultData, argData)

	result, err := json.Marshal(resultData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resultData for filter: %w", err)
	}

	return string(result), nil
}

func (r *Runner) handleOutput(callCtx engine.Context, monitor Monitor, env []string, startState, state *State, retErr error) (*State, error) {
	outputToolRefs, err := callCtx.Tool.GetToolsByType(callCtx.Program, types.ToolTypeOutput)
	if err != nil {
		return nil, err
	}

	if len(outputToolRefs) == 0 {
		return state, retErr
	}

	var (
		continuation bool
		chatFinish   bool
		output       string
	)

	if errMessage := (*engine.ErrChatFinish)(nil); errors.As(retErr, &errMessage) && callCtx.Tool.Chat {
		chatFinish = true
		output = errMessage.Message
	} else if retErr != nil {
		return state, retErr
	} else if state.Continuation != nil && state.Continuation.Result != nil {
		continuation = true
		output = *state.Continuation.Result
	} else if state.Result != nil {
		output = *state.Result
	} else {
		return state, nil
	}

	for _, outputToolRef := range outputToolRefs {
		if callCtx.Program.ToolSet[outputToolRef.ToolID].IsNoop() {
			continue
		}
		inputData, err := argsForFilters(callCtx.Program, outputToolRef, startState, map[string]any{
			"output":       output,
			"continuation": continuation,
			"chat":         callCtx.Tool.Chat,
		})
		if err != nil {
			return nil, fmt.Errorf("marshaling input for output filter: %w", err)
		}
		res, err := r.subCall(callCtx.Ctx, callCtx, monitor, env, outputToolRef.ToolID, inputData, "", engine.OutputToolCategory)
		if err != nil {
			return nil, err
		}
		if res.Result == nil {
			return nil, fmt.Errorf("invalid state: output tool [%s] can not result in a chat continuation", outputToolRef.Reference)
		}
		output = *res.Result
	}

	if chatFinish {
		return state, &engine.ErrChatFinish{
			Message: output,
		}
	} else if continuation {
		state.Continuation.Result = &output
	} else {
		state.Result = &output
	}

	return state, nil
}
