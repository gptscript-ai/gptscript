package credentials

import (
	"github.com/docker/cli/cli/config/credentials"
	"github.com/gptscript-ai/gptscript/pkg/config"
)

type Store struct {
	cfg *config.CLIConfig
}

func NewStore(cfg *config.CLIConfig) (*Store, error) {
	return &Store{cfg: cfg}, nil
}

func (s *Store) Get(toolID string) (*Credential, bool, error) {
	store, err := s.getStore()
	if err != nil {
		return nil, false, err
	}
	auth, err := store.Get(toolID)
	if err != nil {
		return nil, false, err
	} else if auth.Password == "" {
		return nil, false, nil
	}

	cred, err := credentialFromDockerAuthConfig(auth)
	if err != nil {
		return nil, false, err
	}
	return &cred, true, nil
}

func (s *Store) Add(cred Credential) error {
	store, err := s.getStore()
	if err != nil {
		return err
	}
	auth, err := cred.toDockerAuthConfig()
	if err != nil {
		return err
	}
	return store.Store(auth)
}

func (s *Store) Remove(toolID string) error {
	store, err := s.getStore()
	if err != nil {
		return err
	}
	return store.Erase(toolID)
}

func (s *Store) getStore() (credentials.Store, error) {
	return s.getStoreByHelper(config.GPTScriptHelperPrefix + s.cfg.CredentialsStore)
}

func (s *Store) getStoreByHelper(helper string) (credentials.Store, error) {
	if helper == "" || helper == config.GPTScriptHelperPrefix+"file" {
		return credentials.NewFileStore(s.cfg), nil
	}
	return NewHelper(s.cfg, helper)
}
