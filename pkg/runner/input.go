package runner

import (
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/engine"
)

func (r *Runner) handleInput(callCtx engine.Context, monitor Monitor, env []string, input string) (string, error) {
	inputToolRefs, err := callCtx.Tool.GetInputFilterTools(*callCtx.Program)
	if err != nil {
		return "", err
	}

	for _, inputToolRef := range inputToolRefs {
		res, err := r.subCall(callCtx.Ctx, callCtx, monitor, env, inputToolRef.ToolID, input, "", engine.InputToolCategory)
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
