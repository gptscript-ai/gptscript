package sdkserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/loader"
)

type workspaceCommonRequest struct {
	ID            string   `json:"id"`
	WorkspaceTool string   `json:"workspaceTool"`
	Env           []string `json:"env"`
}

func (w workspaceCommonRequest) getToolRepo() string {
	if w.WorkspaceTool != "" {
		return w.WorkspaceTool
	}
	return "github.com/gptscript-ai/workspace-provider"
}

type createWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
	ProviderType           string   `json:"providerType"`
	FromWorkspaceIDs       []string `json:"fromWorkspaceIDs"`
}

func (s *server) createWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject createWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Create Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	if reqObject.ProviderType == "" {
		reqObject.ProviderType = "directory"
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"provider": "%s", "workspace_ids": "%s"}`,
			reqObject.ProviderType, strings.Join(reqObject.FromWorkspaceIDs, ","),
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type deleteWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
}

func (s *server) deleteWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject deleteWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Delete Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s"}`,
			reqObject.ID,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type listWorkspaceContentsRequest struct {
	workspaceCommonRequest `json:",inline"`
	ID                     string `json:"id"`
	Prefix                 string `json:"prefix"`
}

func (s *server) listWorkspaceContents(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject listWorkspaceContentsRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "List Workspace Contents", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "ls_prefix": "%s"}`,
			reqObject.ID, reqObject.Prefix,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type removeAllWithPrefixInWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
	Prefix                 string `json:"prefix"`
}

func (s *server) removeAllWithPrefixInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject removeAllWithPrefixInWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Remove All With Prefix In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "prefix": "%s"}`,
			reqObject.ID, reqObject.Prefix,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type writeFileInWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
	FilePath               string `json:"filePath"`
	Contents               []byte `json:"contents"`
}

func (s *server) writeFileInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject writeFileInWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Write File In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s", "body": "%s"}`,
			reqObject.ID, reqObject.FilePath, base64.StdEncoding.EncodeToString(reqObject.Contents),
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type rmFileInWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
	FilePath               string `json:"filePath"`
}

func (s *server) removeFileInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject rmFileInWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Remove File In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s"}`,
			reqObject.ID, reqObject.FilePath,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type readFileInWorkspaceRequest struct {
	workspaceCommonRequest `json:",inline"`
	FilePath               string `json:"filePath"`
}

func (s *server) readFileInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject readFileInWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Read File In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		reqObject.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s"}`,
			reqObject.ID, reqObject.FilePath,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}
