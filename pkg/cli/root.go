package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/acorn-io/gptscript/pkg/assemble"
	"github.com/acorn-io/gptscript/pkg/input"
	"github.com/acorn-io/gptscript/pkg/loader"
	"github.com/acorn-io/gptscript/pkg/runner"
	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type GPTScript struct {
	runner.Options
	Output   string `usage:"Save output to a file" short:"o"`
	Input    string `usage:"Read input from a file (\"-\" for stdin)" short:"f"`
	SubTool  string `usage:"Target tool name in file to run"`
	Assemble bool   `usage:"Assemble tool to a single artifact and saved to --output"`
}

func New() *cobra.Command {
	return cmd.Command(&GPTScript{})
}

func (r *GPTScript) Customize(cmd *cobra.Command) {
	cmd.Use = version.ProgramName
	cmd.Args = cobra.MinimumNArgs(1)
	cmd.Flags().SetInterspersed(false)
}

func (r *GPTScript) Run(cmd *cobra.Command, args []string) error {
	tool, err := loader.Tool(cmd.Context(), args[0], r.SubTool)
	if err != nil {
		return err
	}

	if r.Assemble {
		var out io.Writer = os.Stdout
		if r.Output != "" {
			f, err := os.Create(r.Output)
			if err != nil {
				return fmt.Errorf("opening %s: %w", r.Output, err)
			}
			defer f.Close()
			out = f
		}

		return assemble.Assemble(cmd.Context(), *tool, out)
	}

	if !r.Quiet {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			r.Quiet = true
		}
	}

	runner, err := runner.New(r.Options)
	if err != nil {
		return err
	}

	toolInput, err := input.FromFile(r.Input)
	if err != nil {
		return err
	}

	if toolInput == "" {
		toolInput = input.FromArgs(args[1:])
	}

	s, err := runner.Run(cmd.Context(), *tool, toolInput)
	if err != nil {
		return err
	}

	if r.Output != "" {
		err = os.WriteFile(r.Output, []byte(s), 0644)
		if err != nil {
			return err
		}
	} else {
		fmt.Print(s)
		if !strings.HasSuffix(s, "\n") {
			fmt.Println()
		}
	}

	return nil
}
