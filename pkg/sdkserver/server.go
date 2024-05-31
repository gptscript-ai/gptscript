package sdkserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/acorn-io/broadcaster"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/rs/cors"
)

type Options struct {
	gptscript.Options

	ListenAddress string
	Debug         bool
}

func Start(ctx context.Context, opts Options) error {
	sigCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer cancel()
	go func() {
		// This is a hack. This server will be run as a forked process in the SDKs. The SDKs will hold stdin open for as long
		// as it wants the server running. When stdin is closed (or the parent process dies), then this will unblock and the
		// server will be shutdown.
		_, _ = io.ReadAll(os.Stdin)
		cancel()
	}()

	if opts.Debug {
		mvl.SetDebug()
	}

	events := broadcaster.New[event]()
	opts.Options.Runner.MonitorFactory = NewSessionFactory(events)
	go events.Start(ctx)

	g, err := gptscript.New(&opts.Options)
	if err != nil {
		return err
	}

	s := &server{
		address:          opts.ListenAddress,
		client:           g,
		events:           events,
		waitingToConfirm: make(map[string]chan runner.AuthorizerResponse),
		waitingToPrompt:  make(map[string]chan map[string]string),
	}
	defer s.Close()

	s.addRoutes(http.DefaultServeMux)

	server := http.Server{
		Addr: opts.ListenAddress,
		Handler: apply(http.DefaultServeMux,
			contentType("application/json"),
			addRequestID,
			addLogger,
			logRequest,
			cors.Default().Handler,
		),
	}

	slog.Info("Starting server", "addr", server.Addr)

	context.AfterFunc(sigCtx, func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		slog.Info("Shutting down server")
		_ = server.Shutdown(ctx)
		slog.Info("Server stopped")
	})

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func (s *server) Close() {
	s.client.Close(true)
	s.events.Close()
}
