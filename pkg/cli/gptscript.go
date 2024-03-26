package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/fatih/color"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/confirm"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/monitor"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type (
	DisplayOptions monitor.Options
	CacheOptions   cache.Options
	OpenAIOptions  openai.Options
)

type GPTScript struct {
	CacheOptions
	OpenAIOptions
	DisplayOptions
	Color         *bool  `usage:"Use color in output (default true)" default:"true"`
	Confirm       bool   `usage:"Prompt before running potentially dangerous commands"`
	Debug         bool   `usage:"Enable debug logging"`
	Quiet         *bool  `usage:"No output logging (set --quiet=false to force on even when there is no TTY)" short:"q"`
	Output        string `usage:"Save output to a file, or - for stdout" short:"o"`
	Input         string `usage:"Read input from a file (\"-\" for stdin)" short:"f"`
	SubTool       string `usage:"Use tool of this name, not the first tool in file"`
	Assemble      bool   `usage:"Assemble tool to a single artifact, saved to --output" hidden:"true"`
	ListModels    bool   `usage:"List the models available and exit"`
	ListTools     bool   `usage:"List built-in tools and exit"`
	Server        bool   `usage:"Start server"`
	ListenAddress string `usage:"Server listen address" default:"127.0.0.1:9090"`
	Chdir         string `usage:"Change current working directory" short:"C"`
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

func (r *GPTScript) Pre(*cobra.Command, []string) error {
	// chdir as soon as possible
	if r.Chdir != "" {
		if err := os.Chdir(r.Chdir); err != nil {
			return err
		}
	}

	if r.DefaultModel != "" {
		builtin.SetDefaultModel(r.DefaultModel)
	}

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
	if r.Color != nil {
		color.NoColor = !*r.Color
	}

	gptOpt := gptscript.Options{
		Cache:   cache.Options(r.CacheOptions),
		OpenAI:  openai.Options(r.OpenAIOptions),
		Monitor: monitor.Options(r.DisplayOptions),
		Quiet:   r.Quiet,
		Env:     os.Environ(),
	}

	if r.Server {
		s, err := server.New(&server.Options{
			ListenAddress: r.ListenAddress,
			GPTScript:     gptOpt,
		})
		if err != nil {
			return err
		}
		defer s.Close()
		return s.Start(cmd.Context())
	}

	gptScript, err := gptscript.New(&gptOpt)
	if err != nil {
		return err
	}
	defer gptScript.Close()

	if r.ListModels {
		models, err := gptScript.ListModels(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(models, "\n"))
		return nil
	}

	if r.ListTools {
		return r.listTools()
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	var (
		prg types.Program
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

	toolInput, err := input.FromCLI(r.Input, args)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	if r.Confirm {
		ctx = confirm.WithConfirm(ctx, confirm.TextPrompt{})
	}
	s, err := gptScript.Run(ctx, prg, os.Environ(), toolInput)
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
