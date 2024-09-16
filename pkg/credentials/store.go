package credentials

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	credentials2 "github.com/docker/docker-credential-helpers/credentials"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"golang.org/x/exp/maps"
)

const (
	DefaultCredentialContext = "default"
	AllCredentialContexts    = "*"
)

type CredentialBuilder interface {
	EnsureCredentialHelpers(ctx context.Context) error
}

type CredentialStore interface {
	Get(ctx context.Context, toolName string) (*Credential, bool, error)
	Add(ctx context.Context, cred Credential) error
	Remove(ctx context.Context, toolName string) error
	List(ctx context.Context) ([]Credential, error)
}

type Store struct {
	credCtx        []string
	credBuilder    CredentialBuilder
	credHelperDirs CredentialHelperDirs
	cfg            *config.CLIConfig
}

func NewStore(cfg *config.CLIConfig, credentialBuilder CredentialBuilder, credCtx, cacheDir string) (CredentialStore, error) {
	if err := validateCredentialCtx(credCtx); err != nil {
		return nil, err
	}
	return Store{
		credCtx:        strings.Split(credCtx, ","),
		credBuilder:    credentialBuilder,
		credHelperDirs: GetCredentialHelperDirs(cacheDir),
		cfg:            cfg,
	}, nil
}

func (s Store) Get(ctx context.Context, toolName string) (*Credential, bool, error) {
	if first(s.credCtx) == AllCredentialContexts {
		return nil, false, fmt.Errorf("cannot get a credential with context %q", AllCredentialContexts)
	}

	store, err := s.getStore(ctx)
	if err != nil {
		return nil, false, err
	}

	var (
		authCfg types.AuthConfig
		credCtx string
	)
	for _, c := range s.credCtx {
		auth, err := store.Get(toolNameWithCtx(toolName, c))
		if err != nil {
			if credentials2.IsErrCredentialsNotFound(err) {
				continue
			}
			return nil, false, err
		} else if auth.Password == "" {
			continue
		}

		authCfg = auth
		credCtx = c
		break
	}

	if credCtx == "" {
		// Didn't find the credential
		return nil, false, nil
	}

	if authCfg.ServerAddress == "" {
		authCfg.ServerAddress = toolNameWithCtx(toolName, credCtx) // Not sure why we have to do this, but we do.
	}

	cred, err := credentialFromDockerAuthConfig(authCfg)
	if err != nil {
		return nil, false, err
	}
	return &cred, true, nil
}

func (s Store) Add(ctx context.Context, cred Credential) error {
	if first(s.credCtx) == AllCredentialContexts {
		return fmt.Errorf("cannot add a credential with context %q", AllCredentialContexts)
	}
	cred.Context = first(s.credCtx)
	store, err := s.getStore(ctx)
	if err != nil {
		return err
	}
	auth, err := cred.toDockerAuthConfig()
	if err != nil {
		return err
	}
	return store.Store(auth)
}

func (s Store) Remove(ctx context.Context, toolName string) error {
	store, err := s.getStore(ctx)
	if err != nil {
		return err
	}

	// TODO - should we erase this cred in all provided contexts, or just the first matching one?
	cred, found, err := s.Get(ctx, toolName)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return store.Erase(toolNameWithCtx(cred.ToolName, cred.Context))
}

func (s Store) List(ctx context.Context) ([]Credential, error) {
	store, err := s.getStore(ctx)
	if err != nil {
		return nil, err
	}
	list, err := store.GetAll()
	if err != nil {
		return nil, err
	}

	credsByContext := make(map[string][]Credential)
	allCreds := make([]Credential, 0)
	for serverAddress, authCfg := range list {
		if authCfg.ServerAddress == "" {
			authCfg.ServerAddress = serverAddress // Not sure why we have to do this, but we do.
		}

		c, err := credentialFromDockerAuthConfig(authCfg)
		if err != nil {
			return nil, err
		}

		allCreds = append(allCreds, c)

		if credsByContext[c.Context] == nil {
			credsByContext[c.Context] = []Credential{c}
		} else {
			credsByContext[c.Context] = append(credsByContext[c.Context], c)
		}
	}

	if first(s.credCtx) == AllCredentialContexts {
		return allCreds, nil
	}

	// Go through the contexts in reverse order so that higher priority contexts override lower ones.
	// TODO - is this how we want to do it?
	credsByName := make(map[string]Credential)
	for i := len(s.credCtx) - 1; i >= 0; i-- {
		for _, c := range credsByContext[s.credCtx[i]] {
			credsByName[c.ToolName] = c
		}
	}

	return maps.Values(credsByName), nil
}

func (s *Store) getStore(ctx context.Context) (credentials.Store, error) {
	return s.getStoreByHelper(ctx, config.GPTScriptHelperPrefix+s.cfg.CredentialsStore)
}

func (s *Store) getStoreByHelper(ctx context.Context, helper string) (credentials.Store, error) {
	if helper == "" || helper == config.GPTScriptHelperPrefix+"file" {
		return credentials.NewFileStore(s.cfg), nil
	}

	// If the helper is referencing one of the credential helper programs, then reference the full path.
	if strings.HasPrefix(helper, "gptscript-credential-") {
		if err := s.credBuilder.EnsureCredentialHelpers(ctx); err != nil {
			return nil, err
		}

		helper = filepath.Join(s.credHelperDirs.BinDir, helper)
	}

	return NewHelper(s.cfg, helper)
}

func validateCredentialCtx(ctx string) error {
	if ctx == "" {
		return fmt.Errorf("credential context cannot be empty")
	}

	if ctx == AllCredentialContexts {
		return nil
	}

	// check alphanumeric
	r := regexp.MustCompile("^[a-zA-Z0-9]+$")
	for _, c := range strings.Split(ctx, ",") {
		if !r.MatchString(c) {
			return fmt.Errorf("credential contexts must be alphanumeric")
		}
	}

	return nil
}
