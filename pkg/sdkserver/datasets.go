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
	Input           string `json:"input"`
	Workspace       string `json:"workspace"`
	DatasetToolRepo string `json:"datasetToolRepo"`
}

func (r datasetRequest) validate(requireInput bool) error {
	if r.Workspace == "" {
		return fmt.Errorf("workspace is required")
	} else if requireInput && r.Input == "" {
		return fmt.Errorf("input is required")
	}
	return nil
}

func (r datasetRequest) opts(o gptscript.Options) gptscript.Options {
	opts := gptscript.Options{
		Cache:     o.Cache,
		Monitor:   o.Monitor,
		Runner:    o.Runner,
		Workspace: r.Workspace,
	}
	return opts
}

func (r datasetRequest) getToolRepo() string {
	if r.DatasetToolRepo != "" {
		return r.DatasetToolRepo
	}
	return "github.com/gptscript-ai/datasets"
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

	prg, err := loader.Program(r.Context(), "List Datasets from "+req.getToolRepo(), "", loader.Options{
		Cache: g.Cache,
	})

	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.gptscriptOpts.Env, req.Input)
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
		return fmt.Errorf("dataset_name is required")
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

	prg, err := loader.Program(r.Context(), "Create Dataset from "+req.getToolRepo(), "", loader.Options{
		Cache: g.Cache,
	})

	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.gptscriptOpts.Env, req.Input)
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
		return fmt.Errorf("dataset_id is required")
	}
	if a.ElementName == "" {
		return fmt.Errorf("element_name is required")
	}
	if a.ElementContent == "" {
		return fmt.Errorf("element_content is required")
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

	prg, err := loader.Program(r.Context(), "Add Element from "+req.getToolRepo(), "", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.gptscriptOpts.Env, req.Input)
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
		return fmt.Errorf("dataset_id is required")
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

	prg, err := loader.Program(r.Context(), "List Elements from "+req.getToolRepo(), "", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.gptscriptOpts.Env, req.Input)
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
		return fmt.Errorf("dataset_id is required")
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

	prg, err := loader.Program(r.Context(), "Get Element from "+req.getToolRepo(), "", loader.Options{
		Cache: g.Cache,
	})
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	result, err := g.Run(r.Context(), prg, s.gptscriptOpts.Env, req.Input)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to run program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": result})
}
