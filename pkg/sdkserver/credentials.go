package sdkserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/gptscript-ai/gptscript/pkg/config"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
)

func (s *server) initializeCredentialStore(ctx context.Context, credCtxs []string) (credentials.CredentialStore, error) {
	cfg, err := config.ReadCLIConfig(s.gptscriptOpts.OpenAI.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CLI config: %w", err)
	}

	if err := s.runtimeManager.SetUpCredentialHelpers(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to set up credential helpers: %w", err)
	}
	if err := s.runtimeManager.EnsureCredentialHelpers(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure credential helpers: %w", err)
	}

	store, err := credentials.NewStore(ctx, cfg, s.runtimeManager, credCtxs, s.gptscriptOpts.Cache.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize credential store: %w", err)
	}

	return store, nil
}

func (s *server) listCredentials(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	req := new(credentialsRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if req.AllContexts {
		req.Context = []string{credentials.AllCredentialContexts}
	} else if len(req.Context) == 0 {
		req.Context = []string{credentials.DefaultCredentialContext}
	}

	store, err := s.initializeCredentialStore(r.Context(), req.Context)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, err)
		return
	}

	creds, err := store.List(r.Context())
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to list credentials: %w", err))
		return
	}

	// Remove the environment variable values (which are secrets) and refresh tokens from the response.
	for i := range creds {
		for k := range creds[i].Env {
			creds[i].Env[k] = ""
		}
		creds[i].RefreshToken = ""
	}

	writeResponse(logger, w, map[string]any{"stdout": creds})
}

func (s *server) createCredential(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	req := new(credentialsRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	cred := new(credentials.Credential)
	if err := json.Unmarshal([]byte(req.Content), cred); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid credential: %w", err))
		return
	}

	if cred.Context == "" {
		cred.Context = credentials.DefaultCredentialContext
	}

	store, err := s.initializeCredentialStore(r.Context(), []string{cred.Context})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, err)
		return
	}

	if err := store.Add(r.Context(), *cred); err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to create credential: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": "Credential created successfully"})
}

func (s *server) revealCredential(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	req := new(credentialsRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if req.Name == "" {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("missing credential name"))
		return
	}

	if req.AllContexts || slices.Contains(req.Context, credentials.AllCredentialContexts) {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("allContexts is not supported for credential retrieval; please specify the specific context that the credential is in"))
		return
	} else if len(req.Context) == 0 {
		req.Context = []string{credentials.DefaultCredentialContext}
	}

	store, err := s.initializeCredentialStore(r.Context(), req.Context)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, err)
		return
	}

	cred, ok, err := store.Get(r.Context(), req.Name)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to get credential: %w", err))
		return
	} else if !ok {
		writeError(logger, w, http.StatusNotFound, fmt.Errorf("credential not found"))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": cred})
}

func (s *server) deleteCredential(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	req := new(credentialsRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
	}

	if req.Name == "" {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("missing credential name"))
		return
	}

	if req.AllContexts || slices.Contains(req.Context, credentials.AllCredentialContexts) {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("allContexts is not supported for credential deletion; please specify the specific context that the credential is in"))
		return
	} else if len(req.Context) == 0 {
		req.Context = []string{credentials.DefaultCredentialContext}
	}

	store, err := s.initializeCredentialStore(r.Context(), req.Context)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, err)
		return
	}

	// Check to see if a cred exists so we can return a 404 if it doesn't.
	if _, ok, err := store.Get(r.Context(), req.Name); err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to get credential: %w", err))
		return
	} else if !ok {
		writeError(logger, w, http.StatusNotFound, fmt.Errorf("credential not found"))
		return
	}

	if err := store.Remove(r.Context(), req.Name); err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to delete credential: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": "Credential deleted successfully"})
}
