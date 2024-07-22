package sdkserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gptscript-ai/broadcaster"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/rs/cors"
)

type Options struct {
	gptscript.Options

	ListenAddress string
	Debug         bool
}

// Run will start the server and block until the server is shut down.
func Run(ctx context.Context, opts Options) error {
	listener, err := newListener(opts)
	if err != nil {
		return err
	}

	_, err = io.WriteString(os.Stderr, listener.Addr().String()+"\n")
	if err != nil {
		return fmt.Errorf("failed to write to address to stderr: %w", err)
	}

	sigCtx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer cancel()
	go func() {
		// This is a hack. This server will be run as a forked process in the SDKs. The SDKs will hold stdin open for as long
		// as it wants the server running. When stdin is closed (or the parent process dies), then this will unblock and the
		// server will be shutdown.
		_, _ = io.ReadAll(os.Stdin)
		cancel()
	}()

	return run(sigCtx, listener, opts)
}

// EmbeddedStart allows running the server as an embedded process that may use Stdin for input.
// It returns the address the server is listening on.
func EmbeddedStart(ctx context.Context, opts Options) (string, error) {
	listener, err := newListener(opts)
	if err != nil {
		return "", err
	}

	go func() {
		_ = run(ctx, listener, opts)
	}()

	return listener.Addr().String(), nil
}

func (s *server) close() {
	s.client.Close(true)
	s.events.Close()
}

func newListener(opts Options) (net.Listener, error) {
	return net.Listen("tcp", opts.ListenAddress)
}

func run(ctx context.Context, listener net.Listener, opts Options) error {
	if opts.Debug {
		mvl.SetDebug()
	}

	events := broadcaster.New[event]()
	opts.Options.Runner.MonitorFactory = NewSessionFactory(events)
	go events.Start(ctx)

	token := uuid.NewString()
	// Add the prompt token env var so that gptscript doesn't start its own server. We never want this client to start the
	// prompt server because it is only used for fmt, parse, etc.
	opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", types.PromptTokenEnvVar, token))

	g, err := gptscript.New(ctx, opts.Options)
	if err != nil {
		return err
	}

	s := &server{
		gptscriptOpts:    opts.Options,
		address:          listener.Addr().String(),
		token:            token,
		client:           g,
		events:           events,
		waitingToConfirm: make(map[string]chan runner.AuthorizerResponse),
		waitingToPrompt:  make(map[string]chan map[string]string),
	}
	defer s.close()

	s.addRoutes(http.DefaultServeMux)

	httpServer := &http.Server{
		Handler: apply(http.DefaultServeMux,
			contentType("application/json"),
			addRequestID,
			addLogger,
			logRequest,
			cors.Default().Handler,
		),
	}

	logger := mvl.Package()
	done := make(chan struct{})
	context.AfterFunc(ctx, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		logger.Infof("Shutting down server")
		_ = httpServer.Shutdown(ctx)
		logger.Infof("Server stopped")
		close(done)
	})

	if err = httpServer.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	<-done
	return nil
}
