//go:build !noyaegi

package engine

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/types"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func (e *Engine) runYaegi(ctx Context, tool types.Tool, input string, toolCategory ToolCategory) (*Return, error) {

	var (
		err error

		instructions []string

		stdOutErrInterpreter bytes.Buffer
		readStd              *os.File
		writeStd             *os.File

		scriptFile string
		scriptData string

		id = counter.Next()
	)

	for _, inputContext := range ctx.InputContext {
		instructions = append(instructions, inputContext.Content)
	}
	envvars := append(e.Env[:], strings.TrimSpace("GPTSCRIPT_CONTEXT="+strings.Join(instructions, "\n")), "GPTSCRIPT_TOOL_DIR="+tool.WorkingDir)
	envvars = appendInputAsEnv(envvars, input)

	interpreterFullLine, scriptData, _ := strings.Cut(tool.Instructions, "\n")
	interpreterAndParams := strings.Fields(interpreterFullLine) // #!sys.yaegi at-syntax.go
	if len(interpreterAndParams) > 1 {
		scriptFile = interpreterAndParams[1] // at-syntax.go
	}

	if scriptFile == "" && strings.TrimSpace(scriptData) == "" {
		return nil, fmt.Errorf("file and data not found at %s", interpreterFullLine)
	}

	if readStd, writeStd, err = os.Pipe(); err != nil {
		return nil, err
	}

	defer func() {
		writeStd.Close()
		readStd.Close()
		stdOutErrInterpreter.Reset()
	}()

	i := interp.New(interp.Options{
		Unrestricted: false,
		Env:          compressEnv(envvars),
		Stdout:       writeStd,
		Stderr:       writeStd,
	})

	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}

	if scriptFile != "" {
		if _, err = i.EvalPath(scriptFile); err != nil {
			return nil, err
		}
	} else if scriptData != "" {
		if _, err = i.Eval(scriptData); err != nil {
			return nil, err
		}
	}

	writeStd.Close()
	_, err = io.Copy(&stdOutErrInterpreter, readStd)
	if err != nil {
		return nil, err
	}

	out := stdOutErrInterpreter.String()
	e.Progress <- types.CompletionStatus{
		CompletionID: id,
		Response: map[string]any{
			"output": out,
			"err":    nil,
		},
	}

	return &Return{
		Result: &out,
	}, nil
}
