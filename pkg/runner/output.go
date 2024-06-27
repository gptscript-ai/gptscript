package runner

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/engine"
)

func (r *Runner) handleOutput(callCtx engine.Context, monitor Monitor, env []string, state *State, retErr error) (*State, error) {
	outputToolRefs, err := callCtx.Tool.GetOutputFilterTools(*callCtx.Program)
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
		inputData, err := json.Marshal(map[string]any{
			"output":       output,
			"continuation": continuation,
			"chat":         callCtx.Tool.Chat,
		})
		if err != nil {
			return nil, fmt.Errorf("marshaling input for output filter: %w", err)
		}
		res, err := r.subCall(callCtx.Ctx, callCtx, monitor, env, outputToolRef.ToolID, string(inputData), "", engine.OutputToolCategory)
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
