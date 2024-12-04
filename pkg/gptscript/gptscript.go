package gptscript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/certs"
	"github.com/gptscript-ai/gptscript/pkg/config"
	context2 "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/llm"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/monitor"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/remote"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"

	// Load all VCS
	_ "github.com/gptscript-ai/gptscript/pkg/loader/vcs"
)

var log = mvl.Package()

type GPTScript struct {
	Registry                  *llm.Registry
	Runner                    *runner.Runner
	Cache                     *cache.Client
	CredentialStoreFactory    credentials.StoreFactory
	DefaultCredentialContexts []string
	WorkspacePath             string
	DeleteWorkspaceOnClose    bool
	ExtraEnv                  []string
	close                     func()
}

type Options struct {
	Cache                cache.Options
	OpenAI               openai.Options
	Monitor              monitor.Options
	Runner               runner.Options
	DefaultModelProvider string
	CredentialContexts   []string
	Quiet                *bool
	Workspace            string
	DisablePromptServer  bool
	SystemToolsDir       string
	Env                  []string
}

func Complete(opts ...Options) Options {
	var result Options
	for _, opt := range opts {
		result.Cache = cache.Complete(result.Cache, opt.Cache)
		result.Monitor = monitor.Complete(result.Monitor, opt.Monitor)
		result.Runner = runner.Complete(result.Runner, opt.Runner)
		result.OpenAI = openai.Complete(result.OpenAI, opt.OpenAI)

		result.SystemToolsDir = types.FirstSet(opt.SystemToolsDir, result.SystemToolsDir)
		result.CredentialContexts = opt.CredentialContexts
		result.Quiet = types.FirstSet(opt.Quiet, result.Quiet)
		result.Workspace = types.FirstSet(opt.Workspace, result.Workspace)
		result.Env = append(result.Env, opt.Env...)
		result.DisablePromptServer = types.FirstSet(opt.DisablePromptServer, result.DisablePromptServer)
		result.DefaultModelProvider = types.FirstSet(opt.DefaultModelProvider, result.DefaultModelProvider)
	}

	if result.Quiet == nil {
		result.Quiet = new(bool)
	}
	if len(result.Env) == 0 {
		result.Env = os.Environ()
	}
	if len(result.CredentialContexts) == 0 {
		result.CredentialContexts = []string{credentials.DefaultCredentialContext}
	}

	return result
}

func New(ctx context.Context, o ...Options) (*GPTScript, error) {
	opts := Complete(o...)
	registry := llm.NewRegistry()

	cacheClient, err := cache.New(opts.Cache)
	if err != nil {
		return nil, err
	}

	cliCfg, err := config.ReadCLIConfig(opts.OpenAI.ConfigFile)
	if err != nil {
		return nil, err
	}

	if opts.Runner.RuntimeManager == nil {
		opts.Runner.RuntimeManager = runtimes.Default(cacheClient.CacheDir(), opts.SystemToolsDir)
	}

	gptscriptCert, err := certs.GenerateGPTScriptCert()
	if err != nil {
		return nil, err
	}

	simplerRunner, err := newSimpleRunner(cacheClient, opts.Runner.RuntimeManager, opts.Env, gptscriptCert)
	if err != nil {
		return nil, err
	}

	storeFactory, err := credentials.NewFactory(ctx, cliCfg, simplerRunner)
	if err != nil {
		return nil, err
	}

	credStore, err := storeFactory.NewStore(opts.CredentialContexts)
	if err != nil {
		return nil, err
	}

	if opts.DefaultModelProvider == "" {
		oaiClient, err := openai.NewClient(ctx, credStore, opts.OpenAI, openai.Options{
			Cache:   cacheClient,
			SetSeed: true,
		})
		if err != nil {
			return nil, err
		}

		if err := registry.AddClient(oaiClient); err != nil {
			return nil, err
		}
	}

	if opts.Runner.MonitorFactory == nil {
		opts.Runner.MonitorFactory = monitor.NewConsole(opts.Monitor, monitor.Options{DebugMessages: *opts.Quiet})
	}

	runner, err := runner.New(registry, credStore, gptscriptCert, opts.Runner)
	if err != nil {
		return nil, err
	}

	var (
		extraEnv    []string
		closeServer = func() {}
	)
	if !opts.DisablePromptServer {
		var ctx context.Context
		ctx, closeServer = context.WithCancel(context2.AddPauseFuncToCtx(context.Background(), opts.Runner.MonitorFactory.Pause))
		extraEnv, err = prompt.NewServer(ctx, opts.Env)
		if err != nil {
			closeServer()
			return nil, err
		}
	}

	fullEnv := append(opts.Env, extraEnv...)

	remoteClient := remote.New(runner, fullEnv, cacheClient, credStore, opts.DefaultModelProvider)
	if err := registry.AddClient(remoteClient); err != nil {
		closeServer()
		return nil, err
	}

	return &GPTScript{
		Registry:                  registry,
		Runner:                    runner,
		Cache:                     cacheClient,
		CredentialStoreFactory:    storeFactory,
		DefaultCredentialContexts: opts.CredentialContexts,
		WorkspacePath:             opts.Workspace,
		DeleteWorkspaceOnClose:    opts.Workspace == "",
		ExtraEnv:                  extraEnv,
		close:                     closeServer,
	}, nil
}

func (g *GPTScript) getEnv(env []string) ([]string, error) {
	var (
		id string
	)

	scheme, rest, isScheme := strings.Cut(g.WorkspacePath, "://")
	if isScheme && scheme == "directory" {
		id = g.WorkspacePath
		g.WorkspacePath = rest
	} else if isScheme {
		id = g.WorkspacePath
		g.WorkspacePath = ""
		g.DeleteWorkspaceOnClose = true
	}

	if g.WorkspacePath == "" {
		var err error
		g.WorkspacePath, err = os.MkdirTemp("", "gptscript-workspace-*")
		if err != nil {
			return nil, err
		}
	} else if !filepath.IsAbs(g.WorkspacePath) {
		var err error
		g.WorkspacePath, err = makeAbsolute(g.WorkspacePath)
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(g.WorkspacePath, 0700); err != nil {
		return nil, err
	}
	if id == "" {
		id = "directory://" + g.WorkspacePath
	}
	return slices.Concat(g.ExtraEnv, env, []string{
		fmt.Sprintf("GPTSCRIPT_WORKSPACE_DIR=%s", g.WorkspacePath),
		fmt.Sprintf("GPTSCRIPT_WORKSPACE_ID=%s", id),
	}), nil
}

func makeAbsolute(path string) (string, error) {
	if strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}

		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return filepath.Abs(path)
}

func (g *GPTScript) Chat(ctx context.Context, prevState runner.ChatState, prg types.Program, envs []string, input string) (runner.ChatResponse, error) {
	envs, err := g.getEnv(envs)
	if err != nil {
		return runner.ChatResponse{}, err
	}

	return g.Runner.Chat(ctx, prevState, prg, envs, input)
}

func (g *GPTScript) Run(ctx context.Context, prg types.Program, envs []string, input string) (string, error) {
	envs, err := g.getEnv(envs)
	if err != nil {
		return "", err
	}

	return g.Runner.Run(ctx, prg, envs, input)
}

func (g *GPTScript) Close(closeDaemons bool) {
	if g.DeleteWorkspaceOnClose && g.WorkspacePath != "" {
		if err := os.RemoveAll(g.WorkspacePath); err != nil {
			log.Errorf("failed to delete workspace %s: %s", g.WorkspacePath, err)
		}
	}

	g.close()

	if closeDaemons {
		engine.CloseDaemons()
	}
}

func (g *GPTScript) GetModel() engine.Model {
	return g.Registry
}

func (g *GPTScript) ListTools(_ context.Context, prg types.Program) []types.Tool {
	if prg.EntryToolID == "" {
		return builtin.ListTools()
	}
	return prg.TopLevelTools()
}

func (g *GPTScript) ListModels(ctx context.Context, providers ...string) ([]string, error) {
	return g.Registry.ListModels(ctx, providers...)
}

type simpleRunner struct {
	cache  *cache.Client
	runner *runner.Runner
	env    []string
}

func newSimpleRunner(cache *cache.Client, rm engine.RuntimeManager, env []string, gptscriptCert certs.CertAndKey) (*simpleRunner, error) {
	runner, err := runner.New(noopModel{}, credentials.NoopStore{}, gptscriptCert, runner.Options{
		RuntimeManager: rm,
		MonitorFactory: simpleMonitorFactory{},
	})
	if err != nil {
		return nil, err
	}
	return &simpleRunner{
		cache:  cache,
		runner: runner,
		env:    env,
	}, nil
}

func (s *simpleRunner) Load(ctx context.Context, toolName string) (prg types.Program, err error) {
	return loader.Program(ctx, toolName, "", loader.Options{
		Cache: s.cache,
	})
}

func (s *simpleRunner) Run(ctx context.Context, prg types.Program, input string) (output string, err error) {
	return s.runner.Run(ctx, prg, s.env, input)
}

type noopModel struct {
}

func (n noopModel) Call(_ context.Context, _ types.CompletionRequest, _ []string, _ chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	return nil, errors.New("unsupported")
}

func (n noopModel) ProxyInfo([]string) (string, string, error) {
	return "", "", errors.New("unsupported")
}

type simpleMonitorFactory struct {
}

func (s simpleMonitorFactory) Start(_ context.Context, _ *types.Program, _ []string, _ string) (runner.Monitor, error) {
	return simpleMonitor{}, nil
}

func (s simpleMonitorFactory) Pause() func() {
	//TODO implement me
	panic("implement me")
}

type simpleMonitor struct {
}

func (s simpleMonitor) Stop(_ context.Context, _ string, _ error) {
}

func (s simpleMonitor) Event(event runner.Event) {
	if event.Type == runner.EventTypeCallProgress {
		if !strings.HasPrefix(event.Content, "{") {
			fmt.Println(event.Content)
		}
	}
}

func (s simpleMonitor) Pause() func() {
	return func() {}
}
