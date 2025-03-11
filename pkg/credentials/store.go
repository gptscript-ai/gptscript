package credentials

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sync"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker-credential-helpers/client"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"golang.org/x/exp/maps"
)

const (
	DefaultCredentialContext = "default"
	AllCredentialContexts    = "*"
)

type CredentialStore interface {
	Get(ctx context.Context, toolName string) (*Credential, bool, error)
	Add(ctx context.Context, cred Credential) error
	Refresh(ctx context.Context, cred Credential) error
	Remove(ctx context.Context, toolName string) error
	List(ctx context.Context) ([]Credential, error)
	RecreateAll(ctx context.Context) error
}

type Store struct {
	credCtxs        []string
	cfg             *config.CLIConfig
	program         client.ProgramFunc
	recreateAllLock sync.RWMutex
}

func (s *Store) Get(_ context.Context, toolName string) (*Credential, bool, error) {
	s.recreateAllLock.RLock()
	defer s.recreateAllLock.RUnlock()

	if len(s.credCtxs) > 0 && s.credCtxs[0] == AllCredentialContexts {
		return nil, false, fmt.Errorf("cannot get a credential with context %q", AllCredentialContexts)
	}

	store, err := s.getStore()
	if err != nil {
		return nil, false, err
	}

	var (
		authCfg types.AuthConfig
		credCtx string
	)
	for _, c := range s.credCtxs {
		auth, err := store.Get(toolNameWithCtx(toolName, c))
		if err != nil {
			if IsCredentialsNotFoundError(err) {
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

// Add adds a new credential to the credential store.
// Any context set on the credential object will be overwritten with the first context of the credential store.
func (s *Store) Add(_ context.Context, cred Credential) error {
	s.recreateAllLock.RLock()
	defer s.recreateAllLock.RUnlock()

	first := first(s.credCtxs)
	if first == AllCredentialContexts {
		return fmt.Errorf("cannot add a credential with context %q", AllCredentialContexts)
	}
	cred.Context = first

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

// Refresh updates an existing credential in the credential store.
func (s *Store) Refresh(_ context.Context, cred Credential) error {
	s.recreateAllLock.RLock()
	defer s.recreateAllLock.RUnlock()

	if !slices.Contains(s.credCtxs, cred.Context) {
		return fmt.Errorf("context %q not in list of valid contexts for this credential store", cred.Context)
	}

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

func (s *Store) Remove(_ context.Context, toolName string) error {
	s.recreateAllLock.RLock()
	defer s.recreateAllLock.RUnlock()

	first := first(s.credCtxs)
	if len(s.credCtxs) > 1 || first == AllCredentialContexts {
		return fmt.Errorf("error: credential deletion is not supported when multiple credential contexts are provided")
	}

	store, err := s.getStore()
	if err != nil {
		return err
	}

	return store.Erase(toolNameWithCtx(toolName, first))
}

func (s *Store) List(_ context.Context) ([]Credential, error) {
	s.recreateAllLock.RLock()
	defer s.recreateAllLock.RUnlock()

	store, err := s.getStore()
	if err != nil {
		return nil, err
	}
	list, err := store.GetAll()
	if err != nil {
		return nil, err
	}

	if len(s.credCtxs) > 0 && s.credCtxs[0] == AllCredentialContexts {
		allCreds := make([]Credential, 0, len(list))
		for serverAddress := range list {
			ac, err := store.Get(serverAddress)
			if err != nil {
				return nil, err
			}
			ac.ServerAddress = serverAddress

			cred, err := credentialFromDockerAuthConfig(ac)
			if err != nil {
				return nil, err
			}
			allCreds = append(allCreds, cred)
		}

		return allCreds, nil
	}

	serverAddressesByContext := make(map[string][]string)
	for serverAddress := range list {
		_, ctx, err := toolNameAndCtxFromAddress(serverAddress)
		if err != nil {
			return nil, err
		}

		if serverAddressesByContext[ctx] == nil {
			serverAddressesByContext[ctx] = []string{serverAddress}
		} else {
			serverAddressesByContext[ctx] = append(serverAddressesByContext[ctx], serverAddress)
		}
	}

	// Go through the contexts in reverse order so that higher priority contexts override lower ones.
	credsByName := make(map[string]Credential)
	for i := len(s.credCtxs) - 1; i >= 0; i-- {
		for _, serverAddress := range serverAddressesByContext[s.credCtxs[i]] {
			ac, err := store.Get(serverAddress)
			if err != nil {
				return nil, err
			}
			ac.ServerAddress = serverAddress

			cred, err := credentialFromDockerAuthConfig(ac)
			if err != nil {
				return nil, err
			}

			toolName, _, err := toolNameAndCtxFromAddress(serverAddress)
			if err != nil {
				return nil, err
			}

			credsByName[toolName] = cred
		}
	}

	return maps.Values(credsByName), nil
}

func (s *Store) RecreateAll(_ context.Context) error {
	store, err := s.getStore()
	if err != nil {
		return err
	}

	// We repeatedly lock and unlock the mutex in this function to give other threads a chance to talk to the credential store.
	// It can take several minutes to recreate the credentials if there are hundreds of them, and we don't want to
	// block all other threads while we do that.
	// New credentials might be created after our GetAll, but they will be created with the current encryption configuration,
	// so it's okay that they are skipped by this function.

	s.recreateAllLock.Lock()
	all, err := store.GetAll()
	s.recreateAllLock.Unlock()
	if err != nil {
		return err
	}

	// Loop through and recreate each individual credential.
	for serverAddress := range all {
		s.recreateAllLock.Lock()
		authConfig, err := store.Get(serverAddress)
		if err != nil {
			s.recreateAllLock.Unlock()

			if IsCredentialsNotFoundError(err) {
				// This can happen if the credential was deleted between the GetAll and the Get by another thread.
				continue
			}
			return err
		}

		if err := store.Erase(serverAddress); err != nil {
			s.recreateAllLock.Unlock()
			return err
		}

		if err := store.Store(authConfig); err != nil {
			s.recreateAllLock.Unlock()
			return err
		}
		s.recreateAllLock.Unlock()
	}

	return nil
}

func (s *Store) getStore() (credentials.Store, error) {
	if s.program != nil {
		return &toolCredentialStore{
			file:    credentials.NewFileStore(s.cfg),
			program: s.program,
		}, nil
	}
	return credentials.NewFileStore(s.cfg), nil
}

func validateCredentialCtx(ctxs []string) error {
	if len(ctxs) == 0 {
		return fmt.Errorf("credential contexts must be provided")
	}

	if len(ctxs) == 1 && ctxs[0] == AllCredentialContexts {
		return nil
	}

	// check alphanumeric
	r := regexp.MustCompile("^[-a-zA-Z0-9.]+$")
	for _, c := range ctxs {
		if !r.MatchString(c) {
			return fmt.Errorf("credential contexts must be alphanumeric")
		}
	}

	return nil
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
