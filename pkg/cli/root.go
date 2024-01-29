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
)

type Root struct {
}

func New() *cobra.Command {
	return cmd.Command(&Root{})
}

func (r *Root) Customize(cmd *cobra.Command) {
	cmd.Use = version.ProgramName
	cmd.Args = cobra.MinimumNArgs(1)
	cmd.Flags().SetInterspersed(false)
}

func (r *Root) Run(cmd *cobra.Command, args []string) error {
	in, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer in.Close()

	mainTool, toolSet, err := parser.Parse(in)
	if err != nil {
		return err
	}

	runner, err := runner.New()
	if err != nil {
		return err
	}

	s, err := runner.Run(cmd.Context(), mainTool, toolSet, "")
	if err != nil {
		return err
	}

	fmt.Print(s)
	if !strings.HasSuffix(s, "\n") {
		fmt.Println()
	}
	return err
}
