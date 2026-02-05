package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/complete"
	"github.com/gptscript-ai/gptscript/pkg/nanobot/uuid"
)

type HTTPServer struct {
	env            map[string]string
	MessageHandler MessageHandler
	sessions       SessionStore
	ctx            context.Context
	healthzPath    string

	// internal health check state
	internalSession *ServerSession
	healthErr       *error
	healthMu        sync.RWMutex
}

type HTTPServerOptions struct {
	SessionStore SessionStore
	BaseContext  context.Context
	HealthzPath  string
}

func (h HTTPServerOptions) Complete() HTTPServerOptions {
	if h.SessionStore == nil {
		h.SessionStore = NewInMemorySessionStore()
	}
	if h.BaseContext == nil {
		h.BaseContext = context.Background()
	}
	return h
}

func (h HTTPServerOptions) Merge(other HTTPServerOptions) (result HTTPServerOptions) {
	h.SessionStore = complete.Last(h.SessionStore, other.SessionStore)
	h.BaseContext = complete.Last(h.BaseContext, other.BaseContext)
	h.HealthzPath = complete.Last(h.HealthzPath, other.HealthzPath)
	return h
}

func NewHTTPServer(env map[string]string, handler MessageHandler, opts ...HTTPServerOptions) *HTTPServer {
	o := complete.Complete(opts...)
	h := &HTTPServer{
		MessageHandler: handler,
		env:            env,
		sessions:       o.SessionStore,
		ctx:            o.BaseContext,
		healthzPath:    o.HealthzPath,
	}

	if h.healthzPath != "" {
		go h.runHealthTicker()
	}

	return h
}

func (h *HTTPServer) streamEvents(rw http.ResponseWriter, req *http.Request) {
	id := h.sessions.ExtractID(req)
	if id == "" {
		id = req.URL.Query().Get("id")
	}

	if id == "" {
		http.Error(rw, "Session ID is required", http.StatusBadRequest)
		return
	}

	session, ok, err := h.sessions.Load(req, id)
	if err != nil {
		http.Error(rw, "Failed to load session: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(rw, "Session not found", http.StatusNotFound)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.WriteHeader(http.StatusOK)
	if flusher, ok := rw.(http.Flusher); ok {
		flusher.Flush()
	}

	session.StartReading()
	defer session.StopReading()

	for {
		msg, ok := session.Read(req.Context())
		if !ok {
			return
		}

		data, _ := json.Marshal(msg)
		_, err := rw.Write([]byte("data: " + string(data) + "\n\n"))
		if err != nil {
			http.Error(rw, "Failed to write message: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func (h *HTTPServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet {
		if h.healthzPath != "" && req.URL.Path == h.healthzPath {
			h.healthMu.RLock()
			healthErr := h.healthErr
			h.healthMu.RUnlock()

			if healthErr == nil {
				http.Error(rw, "waiting for startup", http.StatusTooEarly)
			} else if *healthErr != nil {
				http.Error(rw, (*healthErr).Error(), http.StatusServiceUnavailable)
			} else {
				rw.WriteHeader(http.StatusOK)
			}
			return
		}

		h.streamEvents(rw, req)
		return
	}

	streamingID := h.sessions.ExtractID(req)
	sseID := req.URL.Query().Get("id")

	if streamingID != "" && req.Method == http.MethodDelete {
		sseSession, ok, err := h.sessions.LoadAndDelete(req, streamingID)
		if err != nil {
			http.Error(rw, "Failed to delete session: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(rw, "Session not found", http.StatusNotFound)
			return
		}

		sseSession.Close(true)
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg Message
	if err := json.NewDecoder(req.Body).Decode(&msg); err != nil {
		http.Error(rw, "Failed to decode message: "+err.Error(), http.StatusBadRequest)
		return
	}

	if streamingID != "" {
		streamingSession, ok, err := h.sessions.Load(req, streamingID)
		if err != nil {
			http.Error(rw, "Failed to load session: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(rw, "Session not found", http.StatusNotFound)
			return
		}

		maps.Copy(streamingSession.session.EnvMap(), h.getEnv(req))

		response, err := streamingSession.Exchange(req.Context(), msg)
		if errors.Is(err, ErrNoResponse) {
			rw.WriteHeader(http.StatusAccepted)
			return
		} else if err != nil {
			response = Message{
				JSONRPC: msg.JSONRPC,
				ID:      msg.ID,
				Error:   ErrRPCInternal.WithMessage("%s", err.Error()),
			}
		}

		rw.Header().Set("Content-Type", "application/json")

		if len(response.Result) <= 2 && response.Error == nil && strings.HasPrefix(msg.Method, "notifications/") {
			// Response has no data, write status accepted.
			rw.WriteHeader(http.StatusAccepted)
		}

		if err := json.NewEncoder(rw).Encode(response); err != nil {
			http.Error(rw, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		}

		_ = h.sessions.Store(req, streamingSession.ID(), streamingSession)
		return
	} else if sseID != "" {
		sseSession, ok, err := h.sessions.Load(req, sseID)
		if err != nil {
			http.Error(rw, "Failed to load session: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(rw, "Session not found", http.StatusNotFound)
			return
		}

		maps.Copy(sseSession.session.EnvMap(), h.getEnv(req))

		if err := sseSession.Send(req.Context(), msg); err != nil {
			http.Error(rw, "Failed to handle message: "+err.Error(), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusAccepted)
		return
	}

	if msg.Method != "initialize" {
		http.Error(rw, fmt.Sprintf("Method %q not allowed", msg.Method), http.StatusMethodNotAllowed)
		return
	}

	session, err := NewServerSession(h.ctx, h.MessageHandler)
	if err != nil {
		http.Error(rw, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	maps.Copy(session.session.EnvMap(), h.getEnv(req))

	resp, err := session.Exchange(req.Context(), msg)
	if err != nil {
		http.Error(rw, "Failed to handle message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.sessions.Store(req, session.ID(), session); err != nil {
		http.Error(rw, "Failed to store session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Mcp-Session-Id", session.ID())
	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		http.Error(rw, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *HTTPServer) runHealthTicker() {
	ctx, cancel := context.WithTimeout(h.ctx, 2*time.Minute)
	defer cancel()
	err := h.checkTools(ctx)

	h.healthMu.Lock()
	h.healthErr = &err
	h.healthMu.Unlock()

	timer := time.NewTimer(time.Minute)
	for {
		ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
		err := h.checkTools(ctx)
		cancel()

		h.healthMu.Lock()
		h.healthErr = &err
		h.healthMu.Unlock()

		timer.Reset(time.Minute)
		select {
		case <-h.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (h *HTTPServer) ensureInternalSession(ctx context.Context) (*ServerSession, error) {
	h.healthMu.RLock()
	s := h.internalSession
	h.healthMu.RUnlock()
	if s != nil {
		return s, nil
	}

	session, err := NewServerSession(h.ctx, h.MessageHandler)
	if err != nil {
		return nil, err
	}
	// Set base environment on the internal session
	maps.Copy(session.session.EnvMap(), h.env)

	// Initialize the session
	if _, err := session.Exchange(ctx, Message{
		JSONRPC: "2.0",
		ID:      "healthz-initialize",
		Method:  "initialize",
		Params:  []byte(`{"capabilities":{},"clientInfo":{"name":"nanobot-internal"},"protocolVersion":"2025-06-18"}`),
	}); err != nil {
		session.Close(true)
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	// Send the initialized notification
	if err = session.Send(ctx, Message{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/mcp", nil)
	if err != nil {
		session.Close(true)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	h.sessions.Store(req, session.ID(), session)

	h.healthMu.Lock()
	if s = h.internalSession; s != nil {
		h.healthMu.Unlock()
		// If another goroutine already set the internal session, close this one.
		session.Close(true)
		return s, nil
	}
	h.internalSession = session
	h.healthMu.Unlock()

	return session, nil
}

func (h *HTTPServer) checkTools(ctx context.Context) error {
	session, err := h.ensureInternalSession(ctx)
	if err != nil {
		return err
	}

	resp, err := session.Exchange(ctx, Message{
		JSONRPC: "2.0",
		ID:      uuid.String(),
		Method:  "tools/list",
		Params:  []byte(`{}`),
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var out ListToolsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return fmt.Errorf("failed to parse tools/list result: %w", err)
	}

	if len(out.Tools) == 0 {
		return fmt.Errorf("no tools from server")
	}
	return nil
}

func (h *HTTPServer) getEnv(req *http.Request) map[string]string {
	env := make(map[string]string)
	maps.Copy(env, h.env)
	token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
	if ok {
		env["http:bearer-token"] = token
	}
	for k, v := range req.Header {
		if key, ok := strings.CutPrefix(k, "X-Nanobot-Env-"); ok {
			env[key] = strings.Join(v, ", ")
		}
	}
	return env
}
