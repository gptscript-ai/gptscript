package cli

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/spf13/cobra"
)

type Repl struct {
	gptscript *GPTScript
}

func (e *Repl) Run(cmd *cobra.Command, args []string) error {
	opts, err := e.gptscript.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	runner, err := gptscript.New(&opts)
	if err != nil {
		return err
	}

	ctx := e.gptscript.NewRunContext(cmd)
	for {
		var input string
		err = survey.AskOne(&survey.Input{Message: ">"}, &input)
		if err != nil {
			return err
		}

		if input == "" {
			return nil
		}

		prg, err := loader.Program(ctx, "session.gpt", "")
		if err != nil {
			return err
		}

		toolOutput, err := runner.Run(e.gptscript.NewRunContext(cmd), prg, os.Environ(), input)
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("< " + toolOutput)
	}
}
