package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/expr-lang/expr"

	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runInterpExpr(ctx Context, input string) (*Return, error) {
	// get code to eval
	_, script, _ := strings.Cut(ctx.Tool.Instructions, "\n")
	if script == "" {
		return nil, nil
	}

	// setup environment for the interpreter
	envExpr, err := prepareExprEnv(ctx, input)
	if err != nil {
		return nil, err
	}

	// eval code
	outAny, err := expr.Eval(script, envExpr)
	if err != nil {
		return nil, err
	}
	out := fmt.Sprint(outAny)

	e.Progress <- types.CompletionStatus{
		CompletionID: counter.Next(),
		Response: map[string]any{
			"output": outAny,
			"err":    nil,
		},
	}

	return &Return{
		Result: &out,
	}, nil
}

func prepareExprEnv(engineContext Context, input string) (any, error) {
	// get input
	IN := map[string]any{
		"input": "",
	}
	if input != "" {
		err := json.Unmarshal([]byte(input), &IN)
		if err != nil {
			return nil, err
		}
	}

	// names kept uppercase just for testing to match (kind of) env variables
	return map[string]any{
		"INPUT":             IN["input"],
		"GPTSCRIPT_INPUT":   input,
		"GPTSCRIPT_CONTEXT": &engineContext, // full engine context just to test atm
		// useful stuff from stdlib
		"fmtSprintf": fmt.Sprintf,
	}, nil
}
