package credentials

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/gptscript-ai/gptscript/pkg/config"
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
	credCtx        string
	credBuilder    CredentialBuilder
	credHelperDirs CredentialHelperDirs
	cfg            *config.CLIConfig
}

func NewStore(cfg *config.CLIConfig, credentialBuilder CredentialBuilder, credCtx, cacheDir string) (CredentialStore, error) {
	if err := validateCredentialCtx(credCtx); err != nil {
		return nil, err
	}
	return Store{
		credCtx:        credCtx,
		credBuilder:    credentialBuilder,
		credHelperDirs: GetCredentialHelperDirs(cacheDir),
		cfg:            cfg,
	}, nil
}

func (s Store) Get(ctx context.Context, toolName string) (*Credential, bool, error) {
	store, err := s.getStore(ctx)
	if err != nil {
		return nil, false, err
	}
	auth, err := store.Get(toolNameWithCtx(toolName, s.credCtx))
	if err != nil {
		return nil, false, err
	} else if auth.Password == "" {
		return nil, false, nil
	}

	if auth.ServerAddress == "" {
		auth.ServerAddress = toolNameWithCtx(toolName, s.credCtx) // Not sure why we have to do this, but we do.
	}

	cred, err := credentialFromDockerAuthConfig(auth)
	if err != nil {
		return nil, false, err
	}
	return &cred, true, nil
}

func (s Store) Add(ctx context.Context, cred Credential) error {
	cred.Context = s.credCtx
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
	return store.Erase(toolNameWithCtx(toolName, s.credCtx))
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

	var creds []Credential
	for serverAddress, authCfg := range list {
		if authCfg.ServerAddress == "" {
			authCfg.ServerAddress = serverAddress // Not sure why we have to do this, but we do.
		}

		c, err := credentialFromDockerAuthConfig(authCfg)
		if err != nil {
			return nil, err
		}
		if s.credCtx == "*" || c.Context == s.credCtx {
			creds = append(creds, c)
		}
	}

	return creds, nil
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

	if ctx == "*" { // this represents "all contexts" and is allowed
		return nil
	}

	// check alphanumeric
	r := regexp.MustCompile("^[a-zA-Z0-9]+$")
	if !r.MatchString(ctx) {
		return fmt.Errorf("credential context must be alphanumeric")
	}
	return nil
}
