package sdkserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gptscript-ai/broadcaster"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	gserver "github.com/gptscript-ai/gptscript/pkg/server"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

type server struct {
	gptscriptOpts              gptscript.Options
	address, token             string
	datasetTool, workspaceTool string
	client                     *gptscript.GPTScript
	events                     *broadcaster.Broadcaster[event]

	runtimeManager engine.RuntimeManager

	lock             sync.RWMutex
	waitingToConfirm map[string]chan runner.AuthorizerResponse
	waitingToPrompt  map[string]chan map[string]string
}

func (s *server) addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", s.health)

	mux.HandleFunc("GET /version", s.version)

	// Listing tools supports listing system tools (GET) or listing tools in a gptscript (POST).
	mux.HandleFunc("POST /list-tools", s.listTools)
	mux.HandleFunc("GET /list-tools", s.listTools)
	// Listing models supports listing OpenAI models (GET) or listing models from providers (POST).
	mux.HandleFunc("POST /list-models", s.listModels)
	mux.HandleFunc("GET /list-models", s.listModels)

	mux.HandleFunc("POST /run", s.execHandler)
	mux.HandleFunc("POST /evaluate", s.execHandler)

	mux.HandleFunc("POST /load", s.load)

	mux.HandleFunc("POST /parse", s.parse)
	mux.HandleFunc("POST /fmt", s.fmtDocument)

	mux.HandleFunc("POST /confirm/{id}", s.confirm)
	mux.HandleFunc("POST /prompt/{id}", s.prompt)
	mux.HandleFunc("POST /prompt-response/{id}", s.promptResponse)

	mux.HandleFunc("POST /credentials", s.listCredentials)
	mux.HandleFunc("POST /credentials/create", s.createCredential)
	mux.HandleFunc("POST /credentials/reveal", s.revealCredential)
	mux.HandleFunc("POST /credentials/delete", s.deleteCredential)

	mux.HandleFunc("POST /datasets", s.listDatasets)
	mux.HandleFunc("POST /datasets/create", s.createDataset)
	mux.HandleFunc("POST /datasets/list-elements", s.listDatasetElements)
	mux.HandleFunc("POST /datasets/get-element", s.getDatasetElement)
	mux.HandleFunc("POST /datasets/add-element", s.addDatasetElement)
	mux.HandleFunc("POST /datasets/add-elements", s.addDatasetElements)

	mux.HandleFunc("POST /workspaces/create", s.createWorkspace)
	mux.HandleFunc("POST /workspaces/delete", s.deleteWorkspace)
	mux.HandleFunc("POST /workspaces/list", s.listWorkspaceContents)
	mux.HandleFunc("POST /workspaces/remove-all-with-prefix", s.removeAllWithPrefixInWorkspace)
	mux.HandleFunc("POST /workspaces/write-file", s.writeFileInWorkspace)
	mux.HandleFunc("POST /workspaces/delete-file", s.removeFileInWorkspace)
	mux.HandleFunc("POST /workspaces/read-file", s.readFileInWorkspace)
	mux.HandleFunc("POST /workspaces/stat-file", s.statFileInWorkspace)
}

// health just provides an endpoint for checking whether the server is running and accessible.
func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeResponse(gcontext.GetLogger(r.Context()), w, map[string]string{"stdout": "ok"})
}

// version will return the output of `gptscript --version`
func (s *server) version(w http.ResponseWriter, r *http.Request) {
	writeResponse(gcontext.GetLogger(r.Context()), w, map[string]any{"stdout": fmt.Sprintf("%s version %s", version.ProgramName, version.Get().String())})
}

// listTools will return the output of `gptscript --list-tools`
func (s *server) listTools(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	tools := s.client.ListTools(r.Context(), types.Program{})
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	lines := make([]string, 0, len(tools))
	for _, tool := range tools {
		// Don't print instructions
		tool.Instructions = ""

		lines = append(lines, tool.String())
	}

	writeResponse(logger, w, map[string]any{"stdout": strings.Join(lines, "\n---\n")})
}

// listModels will return the output of `gptscript --list-models`
func (s *server) listModels(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	client := s.client

	var providers []string
	if r.ContentLength != 0 {
		reqObject := new(modelsRequest)
		err := json.NewDecoder(r.Body).Decode(reqObject)
		if err != nil {
			writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
			return
		}

		providers = reqObject.Providers

		client, err = gptscript.New(r.Context(), s.gptscriptOpts, gptscript.Options{Env: reqObject.Env, Runner: runner.Options{CredentialOverrides: reqObject.CredentialOverrides}})
		if err != nil {
			writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to create client: %w", err))
			return
		}
	}

	if s.gptscriptOpts.DefaultModelProvider != "" {
		providers = append(providers, s.gptscriptOpts.DefaultModelProvider)
	}

	out, err := client.ListModels(r.Context(), providers...)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to list models: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": strings.Join(out, "\n")})
}

// execHandler is a general handler for executing tools with gptscript. This is mainly responsible for parsing the request body.
// Then the options and tool are passed to the process function.
func (s *server) execHandler(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to read request body: %w", err))
		return
	}

	reqObject := new(toolOrFileRequest)
	if err := json.Unmarshal(body, reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	ctx := gserver.ContextWithNewRunID(r.Context())
	runID := gserver.RunIDFromContext(ctx)

	// Ensure chat state is not empty.
	if reqObject.ChatState == "" {
		reqObject.ChatState = "null"
	}

	reqObject.Env = append(os.Environ(), reqObject.Env...)
	// Don't overwrite the PromptURLEnvVar if it is already set in the environment.
	var promptTokenAlreadySet bool
	for _, env := range reqObject.Env {
		if v, ok := strings.CutPrefix(env, types.PromptTokenEnvVar+"="); ok && v != "" {
			promptTokenAlreadySet = true
			break
		}
	}
	if !promptTokenAlreadySet {
		// Append a prompt URL for this run.
		reqObject.Env = append(reqObject.Env, fmt.Sprintf("%s=http://%s/prompt/%s", types.PromptURLEnvVar, s.address, runID), fmt.Sprintf("%s=%s", types.PromptTokenEnvVar, s.token))
	}

	logger.Debugf("executing tool: %+v", reqObject)
	var (
		def           fmt.Stringer = &reqObject.ToolDefs
		programLoader              = loaderWithLocation(loader.ProgramFromSource, reqObject.Location)
	)
	if reqObject.Content != "" {
		def = &reqObject.content
	} else if reqObject.File != "" {
		def = &reqObject.file
		programLoader = loader.Program
	}

	opts := gptscript.Options{
		Cache:              cache.Options(reqObject.cacheOptions),
		OpenAI:             openai.Options(reqObject.openAIOptions),
		Env:                reqObject.Env,
		Workspace:          reqObject.Workspace,
		CredentialContexts: reqObject.CredentialContexts,
		Runner: runner.Options{
			// Set the monitor factory so that we can get events from the server.
			MonitorFactory:      NewSessionFactory(s.events),
			CredentialOverrides: reqObject.CredentialOverrides,
			Sequential:          reqObject.ForceSequential,
		},
		DefaultModelProvider: reqObject.DefaultModelProvider,
	}

	if reqObject.Confirm {
		opts.Runner.Authorizer = s.authorize
	}

	s.execAndStream(ctx, programLoader, logger, w, opts, reqObject.ChatState, reqObject.Input, reqObject.SubTool, def)
}

// load will load the file and return the corresponding Program.
func (s *server) load(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	reqObject := new(loadRequest)
	if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	logger.Debugf("parsing file: file=%s, content=%s", reqObject.File, reqObject.Content)

	var (
		prg   types.Program
		err   error
		cache = s.client.Cache
	)

	if reqObject.DisableCache {
		cache = nil
	}

	if reqObject.Content != "" {
		prg, err = loader.ProgramFromSource(r.Context(), reqObject.Content, reqObject.SubTool, loader.Options{Cache: cache})
	} else if reqObject.File != "" {
		prg, err = loader.Program(r.Context(), reqObject.File, reqObject.SubTool, loader.Options{Cache: cache})
	} else {
		prg, err = loader.ProgramFromSource(r.Context(), reqObject.ToolDefs.String(), reqObject.SubTool, loader.Options{Cache: cache})
	}
	if err != nil {
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": map[string]any{"program": prg}})
}

// parse will parse the file and return the corresponding Document.
func (s *server) parse(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	reqObject := new(parseRequest)
	if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	logger.Debugf("parsing file: file=%s, content=%s", reqObject.File, reqObject.Content)

	var (
		out parser.Document
		err error
	)

	if reqObject.Content != "" {
		out, err = parser.Parse(strings.NewReader(reqObject.Content), reqObject.Options)
	} else {
		content, loadErr := input.FromLocation(reqObject.File, reqObject.DisableCache)
		if loadErr != nil {
			logger.Errorf("failed to load file: %v", loadErr)
			writeError(logger, w, http.StatusInternalServerError, loadErr)
			return
		}

		out, err = parser.Parse(strings.NewReader(content), reqObject.Options)
	}
	if err != nil {
		logger.Errorf("failed to parse file: %v", err)
		writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to parse file: %w", err))
		return
	}

	writeResponse(logger, w, map[string]any{"stdout": map[string]any{"nodes": out.Nodes}})
}

// fmtDocument will produce a string representation of the document.
func (s *server) fmtDocument(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	doc := new(parser.Document)
	if err := json.NewDecoder(r.Body).Decode(doc); err != nil {
		writeError(logger, w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
		return
	}

	writeResponse(logger, w, map[string]string{"stdout": doc.String()})
}
