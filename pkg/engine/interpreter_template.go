package engine

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runInterpTemplateText(ctx Context, tool types.Tool, input string) (*Return, error) {
	// get code to eval
	_, script, _ := strings.Cut(ctx.Tool.Instructions, "\n")
	if script == "" {
		return nil, nil
	}

	// setup environment for the interpreter
	tmplData, err := prepareData(&ctx, input)
	if err != nil {
		return nil, err
	}

	tt, err := template.New(tool.Name).Funcs(prepareFuncMap()).Parse(script)
	if err != nil {
		return nil, err
	}

	// eval code
	var tplOut bytes.Buffer
	err = tt.Execute(&tplOut, tmplData)
	if err != nil {
		return nil, err
	}
	out := tplOut.String()

	e.Progress <- types.CompletionStatus{
		CompletionID: counter.Next(),
		Response: map[string]any{
			"output": out,
			"err":    nil,
		},
	}

	return &Return{
		Result: &out,
	}, nil
}

func prepareData(engineContext *Context, input string) (any, error) {
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
		"GPTSCRIPT_CONTEXT": engineContext, // full engine context just to test atm
	}, nil
}

// test helper functions
func prepareFuncMap() template.FuncMap {
	return template.FuncMap{
		"join":       strings.Join,
		"split":      strings.Split,
		"fields":     strings.Fields,
		"trim":       strings.Trim,
		"trimSpace":  strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"cutPrefix": func(s string, prefix string) string {
			after, _ := strings.CutPrefix(s, prefix)
			return after
		},
		"cut": func(s string, sep string) map[string]string {
			before, after, _ := strings.Cut(s, sep)
			return map[string]string{"before": before, "after": after}
		},
		"hasPrefix": strings.HasPrefix,
		"append": func(slice []string, elems ...string) []string {
			return append(slice, elems...)
		},
		"newSlice": func(args ...string) []string {
			if len(args) == 0 {
				return []string{}
			}
			return args
		},
	}
}
