package sdkserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/loader"
)

func (s *server) getDatasetTool(req datasetRequest) string {
	if req.DatasetTool != "" {
		return req.DatasetTool
	}

	return s.datasetTool
}

type datasetRequest struct {
	Input       string   `json:"input"`
	DatasetTool string   `json:"datasetTool"`
	Env         []string `json:"env"`
}

func (r datasetRequest) validate(requireInput bool) error {
	if requireInput && r.Input == "" {
		return fmt.Errorf("input is required")
	} else if len(r.Env) == 0 {
		return fmt.Errorf("env is required")
	}
	return nil
}

func (r datasetRequest) opts(o gptscript.Options) gptscript.Options {
	opts := gptscript.Options{
		Cache:   o.Cache,
		Monitor: o.Monitor,
		Runner:  o.Runner,
	}
	for _, e := range r.Env {
		v, ok := strings.CutPrefix(e, "GPTSCRIPT_WORKSPACE_ID=")
		if ok {
			opts.Workspace = v
		}
	}
	return opts
}

func (s *server) listDatasets(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())

	var req datasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	if err := req.validate(false); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	g, err := gptscript.New(r.Context(), req.opts(s.gptscriptOpts))
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	prg, err := loader.Program(r.Context(), s.getDatasetTool(req), "List Datasets", loader.Options{
		Cache: g.Cache,
	})

	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.getServerToolsEnv(req.Env), req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type addDatasetElementsArgs struct {
	DatasetID   string `json:"datasetID"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Elements    []struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Contents       string `json:"contents"`
		BinaryContents []byte `json:"binaryContents"`
	} `json:"elements"`
}

func (a addDatasetElementsArgs) validate() error {
	if len(a.Elements) == 0 {
		return fmt.Errorf("elements is required")
	}
	return nil
}

func (s *server) addDatasetElements(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())

	var req datasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	if err := req.validate(true); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	g, err := gptscript.New(r.Context(), req.opts(s.gptscriptOpts))
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	var args addDatasetElementsArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), s.getDatasetTool(req), "Add Elements", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.getServerToolsEnv(req.Env), req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type listDatasetElementsArgs struct {
	DatasetID string `json:"datasetID"`
}

func (a listDatasetElementsArgs) validate() error {
	if a.DatasetID == "" {
		return fmt.Errorf("datasetID is required")
	}
	return nil
}

func (s *server) listDatasetElements(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())

	var req datasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	if err := req.validate(true); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	g, err := gptscript.New(r.Context(), req.opts(s.gptscriptOpts))
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	var args listDatasetElementsArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), s.getDatasetTool(req), "List Elements", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.getServerToolsEnv(req.Env), req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type getDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
	Name      string `json:"name"`
}

func (a getDatasetElementArgs) validate() error {
	if a.DatasetID == "" {
		return fmt.Errorf("datasetID is required")
	} else if a.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (s *server) getDatasetElement(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())

	var req datasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
		return
	}

	if err := req.validate(true); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	g, err := gptscript.New(r.Context(), req.opts(s.gptscriptOpts))
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to initialize gptscript: %w", err))
		return
	}
	defer g.Close(false)

	var args getDatasetElementArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), s.getDatasetTool(req), "Get Element", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.getServerToolsEnv(req.Env), req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}
