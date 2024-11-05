package credentials

import (
	"context"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type ProgramLoaderRunner interface {
	Load(ctx context.Context, toolName string) (prg types.Program, err error)
	Run(ctx context.Context, prg types.Program, input string) (output string, err error)
}

func NewFactory(ctx context.Context, cfg *config.CLIConfig, plr ProgramLoaderRunner) (StoreFactory, error) {
	toolName := translateToolName(cfg.CredentialsStore)
	if toolName == config.FileCredHelper {
		return StoreFactory{
			file: true,
			cfg:  cfg,
		}, nil
	}

	prg, err := plr.Load(ctx, toolName)
	if err != nil {
		return StoreFactory{}, err
	}

	return StoreFactory{
		ctx:    ctx,
		prg:    prg,
		runner: plr,
		cfg:    cfg,
	}, nil
}

type StoreFactory struct {
	ctx    context.Context
	prg    types.Program
	file   bool
	runner ProgramLoaderRunner
	cfg    *config.CLIConfig
}

func (s *StoreFactory) NewStore(credCtxs []string) (CredentialStore, error) {
	if err := validateCredentialCtx(credCtxs); err != nil {
		return nil, err
	}
	if s.file {
		return Store{
			credCtxs: credCtxs,
			cfg:      s.cfg,
		}, nil
	}
	return Store{
		credCtxs: credCtxs,
		cfg:      s.cfg,
		program:  s.program,
	}, nil
}

func (s *StoreFactory) program(args ...string) client.Program {
	return &runnerProgram{
		factory: s,
		action:  args[0],
	}
}

func translateToolName(toolName string) string {
	for _, helper := range config.Helpers {
		if helper == toolName {
			return "github.com/gptscript-ai/gptscript-credential-helpers/" + toolName + "/cmd"
		}
	}
	return toolName
}
