package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/gptscript-ai/cmd"
	gptscript2 "github.com/gptscript-ai/go-gptscript"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/auth"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/chat"
	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/loader/github"
	"github.com/gptscript-ai/gptscript/pkg/monitor"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/system"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
	"github.com/gptscript-ai/tui"
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
	Color          *bool  `usage:"Use color in output (default true)" default:"true"`
	Confirm        bool   `usage:"Prompt before running potentially dangerous commands"`
	Debug          bool   `usage:"Enable debug logging"`
	NoTrunc        bool   `usage:"Do not truncate long log messages"`
	Quiet          *bool  `usage:"No output logging (set --quiet=false to force on even when there is no TTY)" short:"q"`
	Output         string `usage:"Save output to a file, or - for stdout" short:"o"`
	EventsStreamTo string `usage:"Stream events to this location, could be a file descriptor/handle (e.g. fd://2), filename, or named pipe (e.g. \\\\.\\pipe\\my-pipe)" name:"events-stream-to"`
	// Input should not be using GPTSCRIPT_INPUT env var because that is the same value that is set in tool executions
	Input                    string   `usage:"Read input from a file (\"-\" for stdin)" short:"f" env:"GPTSCRIPT_INPUT_FILE"`
	SubTool                  string   `usage:"Use tool of this name, not the first tool in file" local:"true"`
	Assemble                 bool     `usage:"Assemble tool to a single artifact, saved to --output" hidden:"true" local:"true"`
	ListModels               bool     `usage:"List the models available and exit" local:"true"`
	ListTools                bool     `usage:"List built-in tools and exit" local:"true"`
	ListenAddress            string   `usage:"Server listen address" default:"127.0.0.1:0" hidden:"true"`
	Chdir                    string   `usage:"Change current working directory" short:"C"`
	Daemon                   bool     `usage:"Run tool as a daemon" local:"true" hidden:"true"`
	Ports                    string   `usage:"The port range to use for ephemeral daemon ports (ex: 11000-12000)" hidden:"true"`
	CredentialContext        string   `usage:"Context name in which to store credentials" default:"default"`
	CredentialOverride       []string `usage:"Credentials to override (ex: --credential-override github.com/example/cred-tool:API_TOKEN=1234)"`
	ChatState                string   `usage:"The chat state to continue, or null to start a new chat and return the state" local:"true"`
	ForceChat                bool     `usage:"Force an interactive chat session if even the top level tool is not a chat tool" local:"true"`
	ForceSequential          bool     `usage:"Force parallel calls to run sequentially" local:"true"`
	Workspace                string   `usage:"Directory to use for the workspace, if specified it will not be deleted on exit"`
	UI                       bool     `usage:"Launch the UI" local:"true" name:"ui"`
	DisableTUI               bool     `usage:"Don't use chat TUI but instead verbose output" local:"true" name:"disable-tui"`
	SaveChatStateFile        string   `usage:"A file to save the chat state to so that a conversation can be resumed with --chat-state" local:"true"`
	DefaultModelProvider     string   `usage:"Default LLM model provider to use, this will override OpenAI settings"`
	GithubEnterpriseHostname string   `usage:"The host name for a Github Enterprise instance to enable for remote loading" local:"true"`

	readData []byte
}

func New() *cobra.Command {
	root := &GPTScript{}
	command := cmd.Command(
		root,
		&Eval{gptscript: root},
		&Credential{root: root},
		&Parse{},
		&Fmt{},
		&Getenv{},
		&SDKServer{
			GPTScript: root,
		},
	)

	// Hide all the global flags for the credential subcommand.
	for _, child := range command.Commands() {
		if strings.HasPrefix(child.Name(), "credential") {
			command.PersistentFlags().VisitAll(func(f *pflag.Flag) {
				newFlag := pflag.Flag{
					Name:  f.Name,
					Usage: f.Usage,
					Value: f.Value,
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
						Value: f.Value,
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

func (r *GPTScript) NewGPTScriptOpts() (gptscript.Options, error) {
	opts := gptscript.Options{
		Cache:   cache.Options(r.CacheOptions),
		OpenAI:  openai.Options(r.OpenAIOptions),
		Monitor: monitor.Options(r.DisplayOptions),
		Runner: runner.Options{
			CredentialOverrides: r.CredentialOverride,
			Sequential:          r.ForceSequential,
		},
		Quiet:                r.Quiet,
		Env:                  os.Environ(),
		CredentialContext:    r.CredentialContext,
		Workspace:            r.Workspace,
		DisablePromptServer:  r.UI,
		DefaultModelProvider: r.DefaultModelProvider,
	}

	if r.Confirm {
		opts.Runner.Authorizer = auth.Authorize
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

	system.SetBinToSelf()

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
		mvl.SetSimpleFormat(!r.NoTrunc)
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
	if r.Output != "" && r.Output != "-" {
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

	if r.GithubEnterpriseHostname != "" {
		loader.AddVSC(github.LoaderForPrefix(r.GithubEnterpriseHostname))
	}

	// If the user is trying to launch the chat-builder UI, then set up the tool and options here.
	if r.UI {
		if os.Getenv(system.BinEnvVar) == "" {
			gptOpt.Env = append(gptOpt.Env, system.BinEnvVar+"="+system.Bin())
		}

		// Pass the corrected environment variables for SDK server options
		if r.DefaultModel != "" {
			gptOpt.Env = append(gptOpt.Env, "GPTSCRIPT_SDKSERVER_DEFAULT_MODEL="+r.DefaultModel)
		}
		if len(r.CredentialOverride) > 0 {
			gptOpt.Env = append(gptOpt.Env, "GPTSCRIPT_SDKSERVER_CREDENTIAL_OVERRIDE="+strings.Join(r.CredentialOverride, ","))
		}

		// If args has more than one element, then the user has provided a file.
		if len(args) > 0 {
			file := args[0]
			if file == "-" {
				return fmt.Errorf("chat UI only supports files, cannot read from stdin")
			}

			// If the file is external, then set the SCRIPTS_PATH to the current working directory. Otherwise,
			// set it to the directory of the script and set the file to the base.
			if !(strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") || strings.HasPrefix(file, "github.com")) {
				absPathToScript, err := filepath.Abs(file)
				if err != nil {
					return fmt.Errorf("cannot determine absolute path to script %s: %v", file, err)
				}
				gptOpt.Env = append(gptOpt.Env, "SCRIPTS_PATH="+filepath.Dir(absPathToScript))
				file = strings.TrimSuffix(filepath.Base(file), ".gpt")
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("could not determine current working directory: %w", err)
				}
				gptOpt.Env = append(gptOpt.Env, "SCRIPTS_PATH="+cwd)
			}

			gptOpt.Env = append(gptOpt.Env, "UI_RUN_FILE="+file)
			// Remove the file from args because the above line will pass it to the UI tool.
			args = args[1:]
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("could not determine current working directory: %w", err)
			}
			gptOpt.Env = append(gptOpt.Env, "SCRIPTS_PATH="+cwd)
		}

		// The UI must run in daemon mode.
		r.Daemon = true
		// Use the UI tool as the first argument.
		args = append([]string{uiTool()}, args...)
	}

	ctx := cmd.Context()

	gptScript, err := gptscript.New(ctx, gptOpt)
	if err != nil {
		return err
	}
	defer gptScript.Close(true)

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

	var chatState string
	if r.ChatState != "" && r.ChatState != "null" && !strings.HasPrefix(r.ChatState, "{") {
		data, err := os.ReadFile(r.ChatState)
		if err != nil {
			return fmt.Errorf("reading %s: %w", r.ChatState, err)
		}
		chatState = string(data)
	}

	// This chat in a stateless mode
	if r.SaveChatStateFile == "-" || r.SaveChatStateFile == "stdout" {
		resp, err := gptScript.Chat(cmd.Context(), chatState, prg, gptOpt.Env, toolInput)
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
		if !r.DisableTUI && !r.Debug && !r.DebugMessages && !r.NoTrunc {
			// Don't use cmd.Context() because then sigint will cancel everything
			return tui.Run(context.Background(), args[0], tui.RunOptions{
				ClientOpts: &gptscript2.GlobalOptions{
					OpenAIAPIKey:         r.OpenAIOptions.APIKey,
					OpenAIBaseURL:        r.OpenAIOptions.BaseURL,
					DefaultModel:         r.DefaultModel,
					DefaultModelProvider: r.DefaultModelProvider,
				},
				TrustedRepoPrefixes: []string{"github.com/gptscript-ai"},
				DisableCache:        r.DisableCache,
				CredentialOverrides: r.CredentialOverride,
				Input:               toolInput,
				CacheDir:            r.CacheDir,
				SubTool:             r.SubTool,
				Workspace:           r.Workspace,
				SaveChatStateFile:   r.SaveChatStateFile,
				ChatState:           chatState,
			})
		}
		return chat.Start(cmd.Context(), chatState, gptScript, func() (types.Program, error) {
			return r.readProgram(ctx, gptScript, args)
		}, gptOpt.Env, toolInput, r.SaveChatStateFile)
	}

	if r.UI {
		// If the UI is running, then all prompts should go through the SDK and the UI.
		// Not clearing ExtraEnv here would mean that the prompts would go through the terminal.
		gptScript.ExtraEnv = nil
	}

	s, err := gptScript.Run(cmd.Context(), prg, gptOpt.Env, toolInput)
	if err != nil {
		return err
	}

	return r.PrintOutput(toolInput, s)
}

// uiTool returns the versioned UI tool reference for the current GPTScript version.
// For release versions, a reference with a matching release tag is returned.
// For all other versions, a reference to main is returned.
func uiTool() string {
	ref := "github.com/gptscript-ai/ui"
	if tag := version.Tag; !strings.Contains(tag, "v0.0.0-dev") {
		ref = fmt.Sprintf("%s@%s", ref, tag)
	}

	return env.VarOrDefault("GPTSCRIPT_CHAT_UI_TOOL", ref)
}
