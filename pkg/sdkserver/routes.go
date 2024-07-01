package sdkserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/broadcaster"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
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

const toolRunTimeout = 15 * time.Minute

type server struct {
	gptscriptOpts  gptscript.Options
	address, token string
	client         *gptscript.GPTScript
	events         *broadcaster.Broadcaster[event]

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

	mux.HandleFunc("POST /parse", s.parse)
	mux.HandleFunc("POST /fmt", s.fmtDocument)

	mux.HandleFunc("POST /confirm/{id}", s.confirm)
	mux.HandleFunc("POST /prompt/{id}", s.prompt)
	mux.HandleFunc("POST /prompt-response/{id}", s.promptResponse)
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
	var prg types.Program
	if r.ContentLength != 0 {
		reqObject := new(toolOrFileRequest)
		err := json.NewDecoder(r.Body).Decode(reqObject)
		if err != nil {
			writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
			return
		}

		if reqObject.Content != "" {
			prg, err = loader.ProgramFromSource(r.Context(), reqObject.Content, reqObject.SubTool, loader.Options{Cache: s.client.Cache})
		} else if reqObject.File != "" {
			prg, err = loader.Program(r.Context(), reqObject.File, reqObject.SubTool, loader.Options{Cache: s.client.Cache})
		} else {
			prg, err = loader.ProgramFromSource(r.Context(), reqObject.ToolDefs.String(), reqObject.SubTool, loader.Options{Cache: s.client.Cache})
		}
		if err != nil {
			writeError(logger, w, http.StatusInternalServerError, fmt.Errorf("failed to load program: %w", err))
			return
		}
	}

	tools := s.client.ListTools(r.Context(), prg)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	lines := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			tool.Name = prg.Name
		}

		// Don't print instructions
		tool.Instructions = ""

		lines = append(lines, tool.String())
	}

	writeResponse(logger, w, map[string]any{"stdout": strings.Join(lines, "\n---\n")})
}

// listModels will return the output of `gptscript --list-models`
func (s *server) listModels(w http.ResponseWriter, r *http.Request) {
	logger := gcontext.GetLogger(r.Context())
	var providers []string
	if r.ContentLength != 0 {
		reqObject := new(modelsRequest)
		if err := json.NewDecoder(r.Body).Decode(reqObject); err != nil {
			writeError(logger, w, http.StatusBadRequest, fmt.Errorf("failed to decode request body: %w", err))
			return
		}

		providers = reqObject.Providers
	}

	out, err := s.client.ListModels(r.Context(), providers...)
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
	ctx, cancel := context.WithTimeout(ctx, toolRunTimeout)
	defer cancel()

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
		programLoader loaderFunc   = loader.ProgramFromSource
	)
	if reqObject.Content != "" {
		def = &reqObject.content
	} else if reqObject.File != "" {
		def = &reqObject.file
		programLoader = loader.Program
	}

	opts := gptscript.Options{
		Cache:             cache.Options(reqObject.cacheOptions),
		OpenAI:            openai.Options(reqObject.openAIOptions),
		Env:               reqObject.Env,
		Workspace:         reqObject.Workspace,
		CredentialContext: reqObject.CredentialContext,
		Runner: runner.Options{
			// Set the monitor factory so that we can get events from the server.
			MonitorFactory:      NewSessionFactory(s.events),
			CredentialOverrides: reqObject.CredentialOverrides,
		},
	}

	if reqObject.Confirm {
		opts.Runner.Authorizer = s.authorize
	}

	s.execAndStream(ctx, programLoader, logger, w, opts, reqObject.ChatState, reqObject.Input, reqObject.SubTool, def)
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
		content, loadErr := input.FromLocation(reqObject.File)
		if loadErr != nil {
			logger.Errorf(loadErr.Error())
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
