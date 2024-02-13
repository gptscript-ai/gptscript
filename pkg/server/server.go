package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/acorn-io/broadcaster"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/static"
	"github.com/olahol/melody"
	"github.com/rs/cors"
)

type Options struct {
	runner.CacheOptions
	runner.OpenAIOptions
	ListenAddress string
}

func complete(opts []Options) (runnerOpts []runner.Options, result Options) {
	for _, opt := range opts {
		result.ListenAddress = types.FirstSet(opt.ListenAddress, result.ListenAddress)
		runnerOpts = append(runnerOpts, runner.Options{
			CacheOptions:  opt.CacheOptions,
			OpenAIOptions: opt.OpenAIOptions,
		})
	}
	if result.ListenAddress == "" {
		result.ListenAddress = "127.0.0.1:9090"
	}
	return
}

func New(opts ...Options) (*Server, error) {
	events := broadcaster.New[Event]()

	runnerOpts, opt := complete(opts)
	r, err := runner.New(append(runnerOpts, runner.Options{
		MonitorFactory: &SessionFactory{
			events: events,
		},
	})...)
	if err != nil {
		return nil, err
	}

	noCacheRunner, err := runner.New(append(runnerOpts, runner.Options{
		CacheOptions: runner.CacheOptions{
			Cache: new(bool),
		},
		MonitorFactory: &SessionFactory{
			events: events,
		},
	})...)
	if err != nil {
		return nil, err
	}

	return &Server{
		melody:        melody.New(),
		events:        events,
		runner:        r,
		noCacheRunner: noCacheRunner,
		listenAddress: opt.ListenAddress,
	}, nil
}

type Event struct {
	runner.Event `json:",inline"`
	RunID        string         `json:"runID,omitempty"`
	Program      *types.Program `json:"program,omitempty"`
	Input        string         `json:"input,omitempty"`
	Output       string         `json:"output,omitempty"`
	Err          string         `json:"err,omitempty"`
}

type Server struct {
	ctx           context.Context
	melody        *melody.Melody
	runner        *runner.Runner
	noCacheRunner *runner.Runner
	events        *broadcaster.Broadcaster[Event]
	listenAddress string
}

var (
	execID int64
)

type execKey struct{}

func (s *Server) list(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	enc.SetIndent("", "  ")

	path := filepath.Join(".", req.URL.Path)
	if req.URL.Path == "/sys" {
		_ = enc.Encode(builtin.SysProgram())
		return
	} else if strings.HasSuffix(path, ".gpt") {
		prg, err := loader.Program(req.Context(), path, req.URL.Query().Get("tool"))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = enc.Encode(prg)
		return
	}

	var result []string
	err := fs.WalkDir(os.DirFS(path), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() && d.Name() != "." {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".gpt") {
			result = append(result, path)
		}

		return nil
	})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = enc.Encode(result)
}

func (s *Server) run(rw http.ResponseWriter, req *http.Request) {
	path := filepath.Join(".", req.URL.Path)
	if !strings.HasSuffix(path, ".gpt") {
		path += ".gpt"
	}

	prg, err := loader.Program(req.Context(), path, req.URL.Query().Get("tool"))
	if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(rw, req)
		return
	} else if err != nil {
		http.Error(rw, err.Error(), http.StatusNotAcceptable)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotAcceptable)
		return
	}

	runner := s.runner
	if req.URL.Query().Has("nocache") {
		runner = s.noCacheRunner
	}

	id := fmt.Sprint(atomic.AddInt64(&execID, 1))
	if req.URL.Query().Has("async") {
		ctx := context.WithValue(s.ctx, execKey{}, id)
		go func() {
			_, _ = runner.Run(ctx, prg, os.Environ(), string(body))
		}()
		rw.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(rw).Encode(map[string]any{
			"id": id,
		})
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	} else {
		ctx := context.WithValue(req.Context(), execKey{}, id)
		out, err := runner.Run(ctx, prg, os.Environ(), string(body))
		if err == nil {
			_, _ = rw.Write([]byte(out))
		} else {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.ctx = ctx
	s.melody.HandleConnect(s.Connect)
	go s.events.Start(ctx)
	log.Infof("Listening on http://%s", s.listenAddress)
	handler := cors.Default().Handler(s)
	server := &http.Server{Addr: s.listenAddress, Handler: handler}
	context.AfterFunc(ctx, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	return server.ListenAndServe()
}

func (s *Server) Connect(session *melody.Session) {
	go func() {
		sub := s.events.Subscribe()
		defer sub.Close()

		for event := range sub.C {
			data, err := json.Marshal(event)
			if err != nil {
				log.Errorf("error marshaling event: %v", err)
				return
			}
			log.Fields("event").Debugf("send")
			err = session.Write(data)
			if err != nil {
				log.Errorf("error writing event: %v", err)
				return
			}
		}
	}()
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	isUpgrade := strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
	isBrowser := strings.Contains(strings.ToLower(req.Header.Get("User-Agent")), "mozilla")
	isAjax := req.Header.Get("X-Requested-With") != ""

	if req.URL.Path == "/" && isBrowser && !isAjax && !isUpgrade {
		rw.Header().Set("Location", "/ui/")
		rw.WriteHeader(302)
		return
	}

	if req.URL.Path == "/favicon.ico" {
		http.ServeFileFS(rw, req, static.UI, "/ui/favicon.ico")
		return
	}

	if strings.HasPrefix(req.URL.Path, "/ui") {
		path := req.URL.Path
		if path == "/ui" || path == "/ui/" {
			path = "/ui/index.html"
		}
		if _, err := fs.Stat(static.UI, path[1:]); errors.Is(err, fs.ErrNotExist) {
			path = "/ui/index.html"
		}
		http.ServeFileFS(rw, req, static.UI, path)
		return
	}

	switch req.Method {
	case http.MethodPost:
		s.run(rw, req)
	case http.MethodGet:
		if req.URL.Path == "/" && isUpgrade {
			err := s.melody.HandleRequest(rw, req)
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
			}
		} else {
			s.list(rw, req)
		}
	default:
		http.NotFound(rw, req)
	}
}

type SessionFactory struct {
	events *broadcaster.Broadcaster[Event]
}

func (s SessionFactory) Start(ctx context.Context, prg *types.Program, env []string, input string) (runner.Monitor, error) {
	id, _ := ctx.Value(execKey{}).(string)

	s.events.C <- Event{
		Event: runner.Event{
			Time: time.Now(),
			Type: "runStart",
		},
		RunID:   id,
		Program: prg,
	}

	return &Session{
		id:     id,
		prj:    prg,
		env:    env,
		input:  input,
		events: s.events,
	}, nil
}

type Session struct {
	id     string
	prj    *types.Program
	env    []string
	input  string
	events *broadcaster.Broadcaster[Event]
}

func (s *Session) Event(event runner.Event) {
	s.events.C <- Event{
		Event: event,
		RunID: s.id,
		Input: s.input,
	}
}

func (s *Session) Stop(output string, err error) {
	e := Event{
		Event: runner.Event{
			Time: time.Now(),
			Type: "runFinish",
		},
		RunID:  s.id,
		Input:  s.input,
		Output: output,
	}
	if err != nil {
		e.Err = err.Error()
	}
	s.events.C <- e
}
