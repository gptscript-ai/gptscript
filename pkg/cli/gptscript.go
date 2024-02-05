package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/acorn-io/gptscript/pkg/assemble"
	"github.com/acorn-io/gptscript/pkg/builtin"
	"github.com/acorn-io/gptscript/pkg/input"
	"github.com/acorn-io/gptscript/pkg/loader"
	"github.com/acorn-io/gptscript/pkg/monitor"
	"github.com/acorn-io/gptscript/pkg/mvl"
	"github.com/acorn-io/gptscript/pkg/openai"
	"github.com/acorn-io/gptscript/pkg/runner"
	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type (
	DisplayOptions monitor.Options
)

type GPTScript struct {
	runner.Options
	DisplayOptions
	Debug      bool   `usage:"Enable debug logging"`
	Quiet      bool   `usage:"No output logging" short:"q"`
	Output     string `usage:"Save output to a file" short:"o"`
	Input      string `usage:"Read input from a file (\"-\" for stdin)" short:"f"`
	SubTool    string `usage:"Use tool of this name, not the first tool in file"`
	Assemble   bool   `usage:"Assemble tool to a single artifact, saved to --output"`
	ListModels bool   `usage:"List the models available and exit"`
	ListTools  bool   `usage:"List built-in tools and exit"`
}

func New() *cobra.Command {
	return cmd.Command(&GPTScript{})
}

func (r *GPTScript) Customize(cmd *cobra.Command) {
	cmd.Use = version.ProgramName + " [flags] PROGRAM_FILE [INPUT...]"
	cmd.Flags().SetInterspersed(false)
}

func (r *GPTScript) listTools(ctx context.Context) error {
	var lines []string
	for _, tool := range builtin.ListTools() {
		lines = append(lines, tool.String())
	}
	fmt.Println(strings.Join(lines, "\n---\n"))
	return nil
}

func (r *GPTScript) listModels(ctx context.Context) error {
	c, err := openai.NewClient(openai.Options(r.OpenAIOptions))
	if err != nil {
		return err
	}

	models, err := c.ListModules(ctx)
	if err != nil {
		return err
	}

	for _, model := range models {
		fmt.Println(model)
	}

	return nil
}

func (r *GPTScript) Pre(cmd *cobra.Command, args []string) error {
	if r.Debug {
		mvl.SetDebug()
	} else {
		mvl.SetSimpleFormat()
	}
	if r.Quiet {
		mvl.SetError()
	}
	return nil
}

func (r *GPTScript) Run(cmd *cobra.Command, args []string) error {
	if r.ListModels {
		return r.listModels(cmd.Context())
	}

	if r.ListTools {
		return r.listTools(cmd.Context())
	}

	if len(args) == 0 {
		return fmt.Errorf("scripts argument required")
	}

	prg, err := loader.Program(cmd.Context(), args[0], r.SubTool)
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

		return assemble.Assemble(cmd.Context(), prg, out)
	}

	if !r.Quiet {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			r.Quiet = true
		}
	}

	runner, err := runner.New(r.Options, runner.Options{
		CacheOptions:   r.CacheOptions,
		OpenAIOptions:  r.OpenAIOptions,
		MonitorFactory: monitor.NewConsole(monitor.Options(r.DisplayOptions)),
	})
	if err != nil {
		return err
	}

	toolInput, err := input.FromCLI(r.Input, args)
	if err != nil {
		return err
	}

	s, err := runner.Run(cmd.Context(), prg, os.Environ(), toolInput)
	if err != nil {
		return err
	}

	if r.Output != "" {
		err = os.WriteFile(r.Output, []byte(s), 0644)
		if err != nil {
			return err
		}
	} else {
		if !r.Quiet {
			if toolInput != "" {
				_, _ = fmt.Fprint(os.Stderr, "\nINPUT:\n\n")
				_, _ = fmt.Fprintln(os.Stderr, toolInput)
			}
			_, _ = fmt.Fprint(os.Stderr, "\nOUTPUT:\n\n")
		}
		fmt.Print(s)
		if !strings.HasSuffix(s, "\n") {
			fmt.Println()
		}
	}

	return nil
}
