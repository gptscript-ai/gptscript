package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/acorn-io/gptscript/pkg/loader"
	"github.com/acorn-io/gptscript/pkg/runner"
	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type GPTScript struct {
	runner.Options
	Output  string `usage:"Save output to a file" short:"o"`
	SubTool string `usage:"Target tool name in file to run"`
}

func New() *cobra.Command {
	return cmd.Command(&GPTScript{})
}

func (r *GPTScript) Customize(cmd *cobra.Command) {
	cmd.Use = version.ProgramName
	cmd.Args = cobra.MinimumNArgs(1)
}

func (r *GPTScript) Run(cmd *cobra.Command, args []string) error {
	tool, err := loader.Tool(cmd.Context(), args[0], r.SubTool)
	if err != nil {
		return err
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

	s, err := runner.Run(cmd.Context(), *tool, "")
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
