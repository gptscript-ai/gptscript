package credentials

import (
	"context"
	"strings"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type ProgramLoaderRunner interface {
	Load(ctx context.Context, toolName string) (prg types.Program, err error)
	Run(ctx context.Context, prg types.Program, input string) (output string, err error)
}

func NewFactory(ctx context.Context, cfg *config.CLIConfig, overrides []string, plr ProgramLoaderRunner) (StoreFactory, error) {
	creds, err := ParseCredentialOverrides(overrides)
	if err != nil {
		return StoreFactory{}, err
	}

	overrideMap := make(map[string]map[string]map[string]string)
	for k, v := range creds {
		contextName, toolName, ok := strings.Cut(k, ctxSeparator)
		if !ok {
			continue
		}
		toolMap, ok := overrideMap[contextName]
		if !ok {
			toolMap = make(map[string]map[string]string)
		}
		toolMap[toolName] = v
		overrideMap[contextName] = toolMap
	}

	toolName := translateToolName(cfg.CredentialsStore)
	if toolName == config.FileCredHelper {
		return StoreFactory{
			file:      true,
			cfg:       cfg,
			overrides: overrideMap,
		}, nil
	}

	prg, err := plr.Load(ctx, toolName)
	if err != nil {
		return StoreFactory{}, err
	}

	return StoreFactory{
		ctx:       ctx,
		prg:       prg,
		runner:    plr,
		cfg:       cfg,
		overrides: overrideMap,
	}, nil
}

type StoreFactory struct {
	ctx    context.Context
	prg    types.Program
	file   bool
	runner ProgramLoaderRunner
	cfg    *config.CLIConfig
	// That's a lot of maps: context -> toolName -> key -> value
	overrides map[string]map[string]map[string]string
}

func (s *StoreFactory) NewStore(credCtxs []string) (CredentialStore, error) {
	if err := validateCredentialCtx(credCtxs); err != nil {
		return nil, err
	}
	if s.file {
		return withOverride{
			target: Store{
				credCtxs: credCtxs,
				cfg:      s.cfg,
			},
			overrides:   s.overrides,
			credContext: credCtxs,
		}, nil
	}
	return withOverride{
		target: Store{
			credCtxs: credCtxs,
			cfg:      s.cfg,
			program:  s.program,
		},
		overrides:   s.overrides,
		credContext: credCtxs,
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
