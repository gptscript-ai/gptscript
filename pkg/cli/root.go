package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/acorn-io/gptscript/pkg/parser"
	"github.com/acorn-io/gptscript/pkg/runner"
	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type GPTScript struct {
	runner.Options
	Output string `usage:"Save output to a file" short:"o"`
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
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	mainTool, toolSet, err := parser.Parse(in)
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

	s, err := runner.Run(cmd.Context(), mainTool, toolSet, "")
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
