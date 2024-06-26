package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/chat"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/spf13/cobra"
)

type Eval struct {
	Tools          []string `usage:"Tools available to call"`
	Chat           bool     `usage:"Enable chat"`
	MaxTokens      int      `usage:"Maximum number of tokens to output"`
	Model          string   `usage:"The model to use"`
	JSON           bool     `usage:"Output JSON"`
	Temperature    string   `usage:"Set the temperature, \"creativity\""`
	InternalPrompt *bool    `Usage:"Set to false to disable the internal prompt"`

	gptscript *GPTScript
}

func (e *Eval) Run(cmd *cobra.Command, args []string) error {
	tool := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description:    "inline script",
				Tools:          e.Tools,
				MaxTokens:      e.MaxTokens,
				ModelName:      e.Model,
				JSONResponse:   e.JSON,
				InternalPrompt: e.InternalPrompt,
				Chat:           e.Chat,
			},
			Instructions: strings.Join(args, " "),
		},
	}

	if e.Temperature != "" {
		temp, err := strconv.ParseFloat(e.Temperature, 32)
		if err != nil {
			return fmt.Errorf("failed to parse %v: %v", e.Temperature, err)
		}
		temp32 := float32(temp)
		tool.Temperature = &temp32
	}

	opts, err := e.gptscript.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	runner, err := gptscript.New(cmd.Context(), opts)
	if err != nil {
		return err
	}

	prg, err := loader.ProgramFromSource(cmd.Context(), tool.String(), "", loader.Options{
		Cache: runner.Cache,
	})
	if err != nil {
		return err
	}

	toolInput, err := input.FromFile(e.gptscript.Input)
	if err != nil {
		return err
	}

	if e.Chat {
		return chat.Start(cmd.Context(), nil, runner, func() (types.Program, error) {
			return prg, nil
		}, os.Environ(), toolInput, "")
	}

	toolOutput, err := runner.Run(cmd.Context(), prg, opts.Env, toolInput)
	if err != nil {
		return err
	}

	return e.gptscript.PrintOutput("", toolOutput)
}
