package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/monitor"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type (
	DisplayOptions monitor.Options
)

type GPTScript struct {
	runner.Options
	DisplayOptions
	Debug         bool   `usage:"Enable debug logging"`
	Quiet         *bool  `usage:"No output logging" short:"q"`
	Output        string `usage:"Save output to a file, or - for stdout" short:"o"`
	Input         string `usage:"Read input from a file (\"-\" for stdin)" short:"f"`
	SubTool       string `usage:"Use tool of this name, not the first tool in file"`
	Assemble      bool   `usage:"Assemble tool to a single artifact, saved to --output"`
	ListModels    bool   `usage:"List the models available and exit"`
	ListTools     bool   `usage:"List built-in tools and exit"`
	Server        bool   `usage:"Start server"`
	ListenAddress string `usage:"Server listen address" default:"127.0.0.1:9090"`
}

func New() *cobra.Command {
	return cmd.Command(&GPTScript{})
}

func (r *GPTScript) Customize(cmd *cobra.Command) {
	cmd.Flags().SetInterspersed(false)
	cmd.Use = version.ProgramName + " [flags] PROGRAM_FILE [INPUT...]"
	cmd.Version = version.Get().String()
	cmd.CompletionOptions.HiddenDefaultCmd = true
	cmd.TraverseChildren = true

	// Enable shell completion for the gptscript command.
	// Note: The gptscript command doesn't have any subcommands, but Cobra requires that at least one is defined before
	// it will generate the completion command automatically. To work around this, define a hidden no-op subcommand.
	cmd.AddCommand(&cobra.Command{Hidden: true})
	cmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Override arg completion to prevent the hidden subcommands from masking default completion for positional args.
	// Note: This should be removed if the gptscript command supports subcommands in the future.
	cmd.ValidArgsFunction = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	}
}

func (r *GPTScript) listTools() error {
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

func (r *GPTScript) Pre(*cobra.Command, []string) error {
	if r.Quiet == nil {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			r.Quiet = new(bool)
		} else {
			r.Quiet = &[]bool{true}[0]
		}
	}

	if r.Debug {
		mvl.SetDebug()
	} else {
		mvl.SetSimpleFormat()
		if *r.Quiet {
			mvl.SetError()
		}
	}
	return nil
}

func (r *GPTScript) Run(cmd *cobra.Command, args []string) error {
	defer engine.CloseDaemons()

	if r.ListModels {
		return r.listModels(cmd.Context())
	}

	if r.ListTools {
		return r.listTools()
	}

	if r.Server {
		s, err := server.New(server.Options{
			CacheOptions:  r.CacheOptions,
			OpenAIOptions: r.OpenAIOptions,
			ListenAddress: r.ListenAddress,
		})
		if err != nil {
			return err
		}
		return s.Start(cmd.Context())
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	var (
		prg types.Program
		err error
	)

	if args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		prg, err = loader.ProgramFromSource(cmd.Context(), string(data), r.SubTool)
		if err != nil {
			return err
		}
	} else {
		prg, err = loader.Program(cmd.Context(), args[0], r.SubTool)
		if err != nil {
			return err
		}
	}

	if r.Assemble {
		var out io.Writer = os.Stdout
		if r.Output != "" && r.Output != "-" {
			f, err := os.Create(r.Output)
			if err != nil {
				return fmt.Errorf("opening %s: %w", r.Output, err)
			}
			defer f.Close()
			out = f
		}

		return assemble.Assemble(prg, out)
	}

	runner, err := runner.New(r.Options, runner.Options{
		CacheOptions:  r.CacheOptions,
		OpenAIOptions: r.OpenAIOptions,
		MonitorFactory: monitor.NewConsole(monitor.Options(r.DisplayOptions), monitor.Options{
			DisplayProgress: !*r.Quiet,
		}),
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
		if !*r.Quiet {
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
