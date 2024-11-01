package sdkserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/loader"
)

type datasetRequest struct {
	Input           string   `json:"input"`
	WorkspaceID     string   `json:"workspaceID"`
	DatasetToolRepo string   `json:"datasetToolRepo"`
	Env             []string `json:"env"`
}

func (r datasetRequest) validate(requireInput bool) error {
	if r.WorkspaceID == "" {
		return fmt.Errorf("workspaceID is required")
	} else if requireInput && r.Input == "" {
		return fmt.Errorf("input is required")
	} else if len(r.Env) == 0 {
		return fmt.Errorf("env is required")
	}
	return nil
}

func (r datasetRequest) opts(o gptscript.Options) gptscript.Options {
	opts := gptscript.Options{
		Cache:     o.Cache,
		Monitor:   o.Monitor,
		Runner:    o.Runner,
		Workspace: r.WorkspaceID,
	}
	return opts
}

func (r datasetRequest) getToolRepo() string {
	if r.DatasetToolRepo != "" {
		return r.DatasetToolRepo
	}
	return "github.com/otto8-ai/datasets"
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

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "List Datasets", loader.Options{
		Cache: g.Cache,
	})

	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type createDatasetArgs struct {
	Name        string `json:"datasetName"`
	Description string `json:"datasetDescription"`
}

func (a createDatasetArgs) validate() error {
	if a.Name == "" {
		return fmt.Errorf("datasetName is required")
	}
	return nil
}

func (s *server) createDataset(w http.ResponseWriter, r *http.Request) {
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

	var args createDatasetArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "Create Dataset", loader.Options{
		Cache: g.Cache,
	})

	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type addDatasetElementArgs struct {
	DatasetID          string `json:"datasetID"`
	ElementName        string `json:"elementName"`
	ElementDescription string `json:"elementDescription"`
	ElementContent     string `json:"elementContent"`
}

func (a addDatasetElementArgs) validate() error {
	if a.DatasetID == "" {
		return fmt.Errorf("datasetID is required")
	}
	if a.ElementName == "" {
		return fmt.Errorf("elementName is required")
	}
	if a.ElementContent == "" {
		return fmt.Errorf("elementContent is required")
	}
	return nil
}

func (s *server) addDatasetElement(w http.ResponseWriter, r *http.Request) {
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

	var args addDatasetElementArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "Add Element", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type addDatasetElementsArgs struct {
	DatasetID string `json:"datasetID"`
	Elements  []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Contents    string `json:"contents"`
	}
}

func (a addDatasetElementsArgs) validate() error {
	if a.DatasetID == "" {
		return fmt.Errorf("datasetID is required")
	}
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

	var args addDatasetElementsArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "Add Elements", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	elementsJSON, err := json.Marshal(args.Elements)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to marshal elements: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, fmt.Sprintf(`{"datasetID":%q, "elements":%q}`, args.DatasetID, string(elementsJSON)))
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

	var args listDatasetElementsArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "List Elements", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}

type getDatasetElementArgs struct {
	DatasetID string `json:"datasetID"`
	Element   string `json:"element"`
}

func (a getDatasetElementArgs) validate() error {
	if a.DatasetID == "" {
		return fmt.Errorf("datasetID is required")
	}
	if a.Element == "" {
		return fmt.Errorf("element is required")
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

	var args getDatasetElementArgs
	if err := json.Unmarshal([]byte(req.Input), &args); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to unmarshal input: %w", err))
		return
	}

	if err := args.validate(); err != nil {
		writeError(logger, w, http.StatusBadRequest, err)
		return
	}

	prg, err := loader.Program(r.Context(), req.getToolRepo(), "Get Element SDK", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, req.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}
