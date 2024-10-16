package sdkserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/loader"
)

type workspaceCommonRequest struct {
	ID            string `json:"id"`
	WorkspaceTool string `json:"workspaceTool"`
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
	DirectoryDataHome      string   `json:"directoryDataHome"`
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
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"provider": "%s", "data_home": "%s", "workspace_ids": "%s"}`,
			reqObject.ProviderType, reqObject.DirectoryDataHome, strings.Join(reqObject.FromWorkspaceIDs, ","),
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
		s.gptscriptOpts.Env,
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
	SubDir                 string `json:"subDir"`
	NonRecursive           bool   `json:"nonRecursive"`
	ExcludeHidden          bool   `json:"excludeHidden"`
	JSON                   bool   `json:"json"`
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
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "ls_sub_dir": "%s", "ls_non_recursive": %t, "ls_exclude_hidden": %t, "ls_json": %t}`,
			reqObject.ID, reqObject.SubDir, reqObject.NonRecursive, reqObject.ExcludeHidden, reqObject.JSON,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type mkDirRequest struct {
	workspaceCommonRequest `json:",inline"`
	DirectoryName          string `json:"directoryName"`
	IgnoreExists           bool   `json:"ignoreExists"`
	CreateDirs             bool   `json:"createDirs"`
}

func (s *server) mkDirInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject mkDirRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Create Directory In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "directory_name": "%s", "mk_dir_ignore_exists": %t, "mk_dir_create_dirs": %t}`,
			reqObject.ID, reqObject.DirectoryName, reqObject.IgnoreExists, reqObject.CreateDirs,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}

type rmDirRequest struct {
	workspaceCommonRequest `json:",inline"`
	DirectoryName          string `json:"directoryName"`
	IgnoreNotFound         bool   `json:"ignoreNotFound"`
	MustBeEmpty            bool   `json:"mustBeEmpty"`
}

func (s *server) rmDirInWorkspace(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var reqObject rmDirRequest
	if err := json.NewDecoder(r.Body).Decode(&reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	prg, err := loader.Program(r.Context(), reqObject.getToolRepo(), "Remove Directory In Workspace", loader.Options{Cache: s.client.Cache})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	out, err := s.client.Run(
		r.Context(),
		prg,
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "directory_name": "%s", "ignore_not_found": %t, "rm_dir_must_be_empty": %t}`,
			reqObject.ID, reqObject.DirectoryName, reqObject.IgnoreNotFound, reqObject.MustBeEmpty,
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
	Contents               string `json:"contents"`
	Base64EncodedInput     bool   `json:"base64EncodedInput"`
	MustNotExist           bool   `json:"mustNotExist"`
	CreateDirs             bool   `json:"createDirs"`
	WithoutCreate          bool   `json:"withoutCreate"`
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
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s", "file_contents": "%s", "write_file_must_not_exist": %t, "write_file_create_dirs": %t, "write_file_without_create": %t, "write_file_base64_encoded_input": %t}`,
			reqObject.ID, reqObject.FilePath, reqObject.Contents, reqObject.MustNotExist, reqObject.CreateDirs, reqObject.WithoutCreate, reqObject.Base64EncodedInput,
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
	IgnoreNotFound         bool   `json:"ignoreNotFound"`
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
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s", "ignore_not_found": %t}`,
			reqObject.ID, reqObject.FilePath, reqObject.IgnoreNotFound,
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
	Base64EncodeOutput     bool   `json:"base64EncodeOutput"`
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
		s.gptscriptOpts.Env,
		fmt.Sprintf(
			`{"workspace_id": "%s", "file_path": "%s", "read_file_base64_encode_output": %t}`,
			reqObject.ID, reqObject.FilePath, reqObject.Base64EncodeOutput,
		),
	)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": out})
}
