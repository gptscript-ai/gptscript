package gptscript

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	context2 "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/llm"
	"github.com/gptscript-ai/gptscript/pkg/monitor"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/remote"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var log = mvl.Package()

type GPTScript struct {
	Registry               *llm.Registry
	Runner                 *runner.Runner
	Cache                  *cache.Client
	WorkspacePath          string
	DeleteWorkspaceOnClose bool
	extraEnv               []string
	close                  func()
}

type Options struct {
	Cache             cache.Options
	OpenAI            openai.Options
	Monitor           monitor.Options
	Runner            runner.Options
	CredentialContext string
	Quiet             *bool
	Workspace         string
	Env               []string
}

func complete(opts *Options) (result *Options) {
	result = opts
	if result == nil {
		result = &Options{}
	}
	if result.Quiet == nil {
		result.Quiet = new(bool)
	}
	if len(result.Env) == 0 {
		result.Env = os.Environ()
	}
	return
}

func New(opts *Options) (*GPTScript, error) {
	opts = complete(opts)

	registry := llm.NewRegistry()

	cacheClient, err := cache.New(opts.Cache)
	if err != nil {
		return nil, err
	}

	oAIClient, err := openai.NewClient(opts.OpenAI, openai.Options{
		Cache:   cacheClient,
		SetSeed: true,
	})
	if err != nil {
		return nil, err
	}

	if err := registry.AddClient(oAIClient); err != nil {
		return nil, err
	}

	if opts.Runner.MonitorFactory == nil {
		opts.Runner.MonitorFactory = monitor.NewConsole(opts.Monitor, monitor.Options{DebugMessages: *opts.Quiet})
	}

	if opts.Runner.RuntimeManager == nil {
		opts.Runner.RuntimeManager = runtimes.Default(cacheClient.CacheDir())
	}

	runner, err := runner.New(registry, opts.CredentialContext, opts.Runner)
	if err != nil {
		return nil, err
	}

	remoteClient := remote.New(runner, opts.Env, cacheClient)

	if err := registry.AddClient(remoteClient); err != nil {
		return nil, err
	}

	ctx, closeServer := context.WithCancel(context2.AddPauseFuncToCtx(context.Background(), opts.Runner.MonitorFactory.Pause))
	extraEnv, err := prompt.NewServer(ctx, opts.Env)
	if err != nil {
		closeServer()
		return nil, err
	}

	return &GPTScript{
		Registry:               registry,
		Runner:                 runner,
		Cache:                  cacheClient,
		WorkspacePath:          opts.Workspace,
		DeleteWorkspaceOnClose: opts.Workspace == "",
		extraEnv:               extraEnv,
		close:                  closeServer,
	}, nil
}

func (g *GPTScript) getEnv(env []string) ([]string, error) {
	if g.WorkspacePath == "" {
		var err error
		g.WorkspacePath, err = os.MkdirTemp("", "gptscript-workspace-*")
		if err != nil {
			return nil, err
		}
	} else if !filepath.IsAbs(g.WorkspacePath) {
		var err error
		g.WorkspacePath, err = filepath.Abs(g.WorkspacePath)
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(g.WorkspacePath, 0700); err != nil {
		return nil, err
	}
	return slices.Concat(g.extraEnv, []string{
		fmt.Sprintf("GPTSCRIPT_WORKSPACE_DIR=%s", g.WorkspacePath),
		fmt.Sprintf("GPTSCRIPT_WORKSPACE_ID=%s", hash.ID(g.WorkspacePath)),
	}, env), nil
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
