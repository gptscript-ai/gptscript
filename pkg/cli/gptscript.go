package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/acorn-io/cmd"
	"github.com/fatih/color"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/chat"
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
	"github.com/spf13/pflag"
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
	Color              *bool  `usage:"Use color in output (default true)" default:"true"`
	Confirm            bool   `usage:"Prompt before running potentially dangerous commands"`
	Debug              bool   `usage:"Enable debug logging"`
	Quiet              *bool  `usage:"No output logging (set --quiet=false to force on even when there is no TTY)" short:"q"`
	Output             string `usage:"Save output to a file, or - for stdout" short:"o"`
	EventsStreamTo     string `usage:"Stream events to this location, could be a file descriptor/handle (e.g. fd://2), filename, or named pipe (e.g. \\\\.\\pipe\\my-pipe)" name:"events-stream-to"`
	Input              string `usage:"Read input from a file (\"-\" for stdin)" short:"f"`
	SubTool            string `usage:"Use tool of this name, not the first tool in file" local:"true"`
	Assemble           bool   `usage:"Assemble tool to a single artifact, saved to --output" hidden:"true" local:"true"`
	ListModels         bool   `usage:"List the models available and exit" local:"true"`
	ListTools          bool   `usage:"List built-in tools and exit" local:"true"`
	Server             bool   `usage:"Start server" local:"true"`
	ListenAddress      string `usage:"Server listen address" default:"127.0.0.1:9090" local:"true"`
	Chdir              string `usage:"Change current working directory" short:"C"`
	Daemon             bool   `usage:"Run tool as a daemon" local:"true" hidden:"true"`
	Ports              string `usage:"The port range to use for ephemeral daemon ports (ex: 11000-12000)" hidden:"true"`
	CredentialContext  string `usage:"Context name in which to store credentials" default:"default"`
	CredentialOverride string `usage:"Credentials to override (ex: --credential-override github.com/example/cred-tool:API_TOKEN=1234)"`
	ChatState          string `usage:"The chat state to continue, or null to start a new chat and return the state"`
	ForceChat          bool   `usage:"Force an interactive chat session if even the top level tool is not a chat tool"`
	Workspace          string `usage:"Directory to use for the workspace, if specified it will not be deleted on exit"`

	readData []byte
}

func New() *cobra.Command {
	root := &GPTScript{}
	command := cmd.Command(
		root,
		&Eval{
			gptscript: root,
		},
		&Credential{root: root},
		&Parse{},
		&Fmt{},
	)

	// Hide all the global flags for the credential subcommand.
	for _, child := range command.Commands() {
		if strings.HasPrefix(child.Name(), "credential") {
			command.PersistentFlags().VisitAll(func(f *pflag.Flag) {
				newFlag := pflag.Flag{
					Name:  f.Name,
					Usage: f.Usage,
				}

				if f.Name != "credential-context" { // We want to keep credential-context
					child.Flags().AddFlag(&newFlag)
					child.Flags().Lookup(newFlag.Name).Hidden = true
				}
			})

			for _, grandchild := range child.Commands() {
				command.PersistentFlags().VisitAll(func(f *pflag.Flag) {
					newFlag := pflag.Flag{
						Name:  f.Name,
						Usage: f.Usage,
					}

					if f.Name != "credential-context" {
						grandchild.Flags().AddFlag(&newFlag)
						grandchild.Flags().Lookup(newFlag.Name).Hidden = true
					}
				})
			}

			break
		}
	}

	return command
}

func (r *GPTScript) NewRunContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if r.Confirm {
		ctx = confirm.WithConfirm(ctx, confirm.TextPrompt{})
	}
	return ctx
}

func (r *GPTScript) NewGPTScriptOpts() (gptscript.Options, error) {
	opts := gptscript.Options{
		Cache:             cache.Options(r.CacheOptions),
		OpenAI:            openai.Options(r.OpenAIOptions),
		Monitor:           monitor.Options(r.DisplayOptions),
		Quiet:             r.Quiet,
		Env:               os.Environ(),
		CredentialContext: r.CredentialContext,
		Workspace:         r.Workspace,
	}

	if r.Ports != "" {
		start, end, _ := strings.Cut(r.Ports, "-")
		startNum, err := strconv.ParseInt(strings.TrimSpace(start), 10, 64)
		if err != nil {
			return gptscript.Options{}, fmt.Errorf("invalid port range: %s", r.Ports)
		}
		var endNum int64
		if end != "" {
			endNum, err = strconv.ParseInt(strings.TrimSpace(end), 10, 64)
			if err != nil {
				return gptscript.Options{}, fmt.Errorf("invalid port range: %s", r.Ports)
			}
		}
		opts.Runner.StartPort = startNum
		opts.Runner.EndPort = endNum
	}

	opts.Runner.CredentialOverride = r.CredentialOverride

	if r.EventsStreamTo != "" {
		mf, err := monitor.NewFileFactory(r.EventsStreamTo)
		if err != nil {
			return gptscript.Options{}, err
		}

		opts.Runner.MonitorFactory = mf
	}

	return opts, nil
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

func (r *GPTScript) listTools(ctx context.Context, gptScript *gptscript.GPTScript, prg types.Program) error {
	tools := gptScript.ListTools(ctx, prg)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	var lines []string
	for _, tool := range tools {
		if tool.Name == "" {
			tool.Name = prg.Name
		}

		// Don't print instructions
		tool.Instructions = ""

		lines = append(lines, tool.String())
	}
	fmt.Println(strings.Join(lines, "\n---\n"))
	return nil
}

func (r *GPTScript) PersistentPre(*cobra.Command, []string) error {
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
			if r.Color == nil {
				r.Color = new(bool)
			}
		}
	}

	if r.Debug {
		mvl.SetDebug()
		if r.Color == nil {
			r.Color = new(bool)
		}
	} else {
		mvl.SetSimpleFormat()
		if *r.Quiet {
			mvl.SetError()
		}
	}

	if r.Color != nil {
		color.NoColor = !*r.Color
	}

	if r.DefaultModel != openai.DefaultModel {
		log.Infof("WARNING: Changing the default model can have unknown behavior for existing tools. Use the model field per tool instead.")
	}

	return nil
}

func (r *GPTScript) listModels(ctx context.Context, gptScript *gptscript.GPTScript, args []string) error {
	models, err := gptScript.ListModels(ctx, args...)
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(models, "\n"))
	return nil
}

func (r *GPTScript) readProgram(ctx context.Context, runner *gptscript.GPTScript, args []string) (prg types.Program, err error) {
	if len(args) == 0 {
		return
	}

	if args[0] == "-" {
		var (
			data []byte
			err  error
		)
		if len(r.readData) > 0 {
			data = r.readData
		} else {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return prg, err
			}
			r.readData = data
		}
		return loader.ProgramFromSource(ctx, string(data), r.SubTool, loader.Options{
			Cache: runner.Cache,
		})
	}

	return loader.Program(ctx, args[0], r.SubTool, loader.Options{
		Cache: runner.Cache,
	})
}

func (r *GPTScript) PrintOutput(toolInput, toolOutput string) (err error) {
	if r.Output != "" {
		err = os.WriteFile(r.Output, []byte(toolOutput), 0644)
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
		fmt.Print(toolOutput)
		if !strings.HasSuffix(toolOutput, "\n") {
			fmt.Println()
		}
	}

	return
}

func (r *GPTScript) Run(cmd *cobra.Command, args []string) (retErr error) {
	gptOpt, err := r.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	if r.Server {
		s, err := server.New(&server.Options{
			ListenAddress: r.ListenAddress,
			GPTScript:     gptOpt,
		})
		if err != nil {
			return err
		}
		defer s.Close()
		return s.Start(ctx)
	}

	gptScript, err := gptscript.New(&gptOpt)
	if err != nil {
		return err
	}
	defer gptScript.Close()

	if r.ListModels {
		return r.listModels(ctx, gptScript, args)
	}

	prg, err := r.readProgram(ctx, gptScript, args)
	if err != nil {
		return err
	}

	if r.Daemon {
		prg = prg.SetBlocking()
		defer func() {
			if retErr == nil {
				<-ctx.Done()
			}
		}()
	}

	if r.ListTools {
		return r.listTools(ctx, gptScript, prg)
	}

	if len(args) == 0 {
		return cmd.Help()
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

	if r.ChatState != "" {
		resp, err := gptScript.Chat(r.NewRunContext(cmd), r.ChatState, prg, os.Environ(), toolInput)
		if err != nil {
			return err
		}
		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		return r.PrintOutput(toolInput, string(data))
	}

	if prg.IsChat() || r.ForceChat {
		return chat.Start(r.NewRunContext(cmd), nil, gptScript, func() (types.Program, error) {
			return r.readProgram(ctx, gptScript, args)
		}, os.Environ(), toolInput)
	}

	s, err := gptScript.Run(r.NewRunContext(cmd), prg, os.Environ(), toolInput)
	if err != nil {
		return err
	}

	return r.PrintOutput(toolInput, s)
}
