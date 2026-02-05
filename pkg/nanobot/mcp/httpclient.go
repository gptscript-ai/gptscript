package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/log"
	"github.com/gptscript-ai/gptscript/pkg/nanobot/uuid"
)

const SessionIDHeader = "Mcp-Session-Id"

type HTTPClient struct {
	ctx          context.Context
	cancel       context.CancelFunc
	clientLock   sync.RWMutex
	httpClient   *http.Client
	handler      WireHandler
	oauthHandler *oauth
	baseURL      string
	messageURL   string
	serverName   string
	headers      map[string]string
	waiter       *waiter
	sse          bool

	initializeLock    sync.RWMutex
	initializeRequest *Message
	sessionID         *string

	sseLock       sync.RWMutex
	needReconnect bool
}

func newHTTPClient(serverName, baseURL, oauthClientName, oauthRedirectURL string, callbackHandler CallbackHandler, clientCredLookup ClientCredLookup, tokenStorage TokenStorage, headers map[string]string) *HTTPClient {
	var sessionID *string
	if id := headers[SessionIDHeader]; id != "" {
		sessionID = &id
	}
	h := &HTTPClient{
		httpClient:    http.DefaultClient,
		oauthHandler:  newOAuth(callbackHandler, clientCredLookup, tokenStorage, oauthClientName, oauthRedirectURL),
		baseURL:       baseURL,
		messageURL:    baseURL,
		serverName:    serverName,
		headers:       maps.Clone(headers),
		waiter:        newWaiter(),
		needReconnect: true,
		sessionID:     sessionID,
	}

	return h
}

func (s *HTTPClient) SetOAuthCallbackHandler(handler CallbackHandler) {
	s.oauthHandler.callbackHandler = handler
}

func (s *HTTPClient) SessionID() string {
	s.initializeLock.RLock()
	defer s.initializeLock.RUnlock()

	if s.sessionID == nil {
		return ""
	}
	return *s.sessionID
}

func (s *HTTPClient) Close(deleteSession bool) {
	if deleteSession {
		s.initializeLock.RLock()
		sessionID := s.sessionID
		s.initializeLock.RUnlock()

		if sessionID != nil && *sessionID != "" {
			// If we have a session ID, then we need to send a close message to
			// the server to clean up the session.
			s.clientLock.RLock()
			httpClient := s.httpClient
			s.clientLock.RUnlock()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := s.newRequest(ctx, http.MethodDelete, nil)
			if err != nil {
				log.Errorf(ctx, "failed to create close request: %v", err)
				return
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				// Best effort
				log.Errorf(ctx, "failed to send close request: %v", err)
				return
			}
			resp.Body.Close()
		}
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.waiter.Close()
}

func (s *HTTPClient) Wait() {
	s.waiter.Wait()
}

func (s *HTTPClient) newRequest(ctx context.Context, method string, in any) (*http.Request, error) {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		body = bytes.NewBuffer(data)
		log.Messages(ctx, s.serverName, true, data)
	}

	u := s.messageURL
	if method == http.MethodGet || u == "" {
		// If this is a GET request, then it is starting the SSE stream.
		// In this case, we need to use the base URL instead.
		u = s.baseURL
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}

	for k, v := range s.headers {
		req.Header.Set(k, v)
	}

	s.initializeLock.RLock()
	if s.sessionID != nil && *s.sessionID != "" {
		req.Header.Set(SessionIDHeader, *s.sessionID)
	}
	s.initializeLock.RUnlock()

	req.Header.Set("Accept", "text/event-stream")
	if method != http.MethodGet {
		// Don't add because some *cough* CloudFront *cough* proxies don't like it
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (s *HTTPClient) ensureSSE(ctx context.Context, msg *Message, lastEventID any) error {
	s.sseLock.RLock()
	if !s.needReconnect {
		s.sseLock.RUnlock()
		return nil
	}
	s.sseLock.RUnlock()

	// Hold the lock while we try to start the SSE endpoint.
	// We need to make sure that the message URL is set before continuing.
	s.sseLock.Lock()
	defer s.sseLock.Unlock()

	if !s.needReconnect {
		// Check again in case SSE was started while we were waiting for the lock.
		return nil
	}

	// Start the SSE stream with the managed context.
	req, err := s.newRequest(s.ctx, http.MethodGet, nil)
	if err != nil {
		return err
	}

	if lastEventID != nil {
		req.Header.Set("Last-Event-ID", fmt.Sprintf("%v", lastEventID))
	}

	s.clientLock.RLock()
	httpClient := s.httpClient
	s.clientLock.RUnlock()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return AuthRequiredErr{
			ProtectedResourceValue: resp.Header.Get("WWW-Authenticate"),
			Err:                    fmt.Errorf("failed to connect to SSE server: %s: %s", resp.Status, body),
		}
	}

	if resp.StatusCode == http.StatusNotFound && !s.sse {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		s.initializeLock.RLock()
		defer s.initializeLock.RUnlock()

		return SessionNotFoundErr{
			SessionID: *s.sessionID,
			Err:       fmt.Errorf("failed to connect to SSE server: %s: %s", resp.Status, body),
		}
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		// If msg is nil, then this is an SSE request for HTTP streaming.
		// If the server doesn't support a separate SSE endpoint, then we can just return.
		if !s.sse && resp.StatusCode == http.StatusMethodNotAllowed {
			s.needReconnect = false
			return nil
		}
		return fmt.Errorf("failed to connect to SSE server url %s: %s, %s", req.URL.String(), resp.Status, string(body))
	}

	s.needReconnect = false

	gotResponse := make(chan error, 1)
	go func() (err error, send bool) {
		defer func() {
			if err != nil {
				s.sseLock.Lock()
				s.needReconnect = true
				s.sseLock.Unlock()

				// If we get an error, then we aren't reconnecting to the SSE endpoint.
				if send {
					gotResponse <- err
				}
			}

			resp.Body.Close()
		}()

		messages := newSSEStream(resp.Body)

		if s.sse {
			data, ok := messages.readNextMessage()
			if !ok {
				return fmt.Errorf("failed to read SSE message: %w", messages.err()), true
			}

			baseURL, err := url.Parse(s.baseURL)
			if err != nil {
				return fmt.Errorf("failed to parse SSE URL: %w", err), true
			}

			u, err := url.Parse(data)
			if err != nil {
				return fmt.Errorf("failed to parse returned SSE URL: %w", err), true
			}

			baseURL.Path = u.Path
			baseURL.RawQuery = u.RawQuery
			s.messageURL = baseURL.String()

			initReq, err := s.newRequest(ctx, http.MethodPost, msg)
			if err != nil {
				return fmt.Errorf("failed to create initialize message req: %w", err), true
			}

			s.clientLock.RLock()
			httpClient = s.httpClient
			s.clientLock.RUnlock()

			initResp, err := httpClient.Do(initReq)
			if err != nil {
				return fmt.Errorf("failed to POST initialize message: %w", err), true
			}
			body, _ := io.ReadAll(initResp.Body)
			_ = initResp.Body.Close()

			if initResp.StatusCode != http.StatusOK && initResp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("failed to POST initialize message got status: %s: %s", initResp.Status, body), true
			}

			// Mark this client as initialized.
			s.initializeLock.Lock()
			s.sessionID = new(string)
			s.initializeRequest = msg
			s.initializeLock.Unlock()
		}

		close(gotResponse)

		for {
			message, ok := messages.readNextMessage()
			if !ok {
				if err := messages.err(); err != nil {
					if errors.Is(err, context.Canceled) {
						log.Debugf(ctx, "context canceled reading SSE message: %v", messages.err())
					} else {
						log.Errorf(ctx, "failed to read SSE message: %v", messages.err())
					}
				}

				select {
				case <-s.ctx.Done():
					// If the context is done, then we don't need to reconnect.
					// Returning the error here will close the waiter, indicating that
					// the client is done.
					return s.ctx.Err(), false
				default:
					if msg != nil {
						msg.ID = uuid.String()
					}
					s.sseLock.Lock()
					if !s.needReconnect {
						s.needReconnect = true
					}
					s.sseLock.Unlock()
				}

				if err := s.ensureSSE(ctx, msg, lastEventID); err != nil {
					return fmt.Errorf("failed to reconnect to SSE server: %v", err), false
				}

				return nil, false
			}

			var msg Message
			if err := json.Unmarshal([]byte(message), &msg); err != nil {
				continue
			}

			if msg.ID != nil {
				lastEventID = msg.ID
			}

			log.Messages(ctx, s.serverName, false, []byte(message))
			s.handler(s.ctx, msg)
		}
	}()

	return <-gotResponse
}

func (s *HTTPClient) Start(ctx context.Context, handler WireHandler) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.handler = handler

	if httpClient := s.oauthHandler.loadFromStorage(s.ctx, s.baseURL); httpClient != nil {
		s.httpClient = httpClient
	}

	return nil
}

func (s *HTTPClient) initialize(ctx context.Context, msg Message) error {
	req, err := s.newRequest(ctx, http.MethodPost, msg)
	if err != nil {
		return err
	}

	s.clientLock.RLock()
	httpClient := s.httpClient
	s.clientLock.RUnlock()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		streamingErrorMessage, _ := io.ReadAll(resp.Body)
		return AuthRequiredErr{
			ProtectedResourceValue: resp.Header.Get("WWW-Authenticate"),
			Err:                    fmt.Errorf("failed to initialize HTTP Streaming client: %s: %s", resp.Status, streamingErrorMessage),
		}
	}

	if resp.StatusCode != http.StatusOK {
		streamingErrorMessage, _ := io.ReadAll(resp.Body)
		streamError := fmt.Errorf("failed to initialize HTTP Streaming client: %s: %s", resp.Status, streamingErrorMessage)

		s.sse = true
		if err := s.ensureSSE(ctx, &msg, nil); err != nil {
			s.sse = false
			return errors.Join(streamError, err)
		}

		// The client is marked as initialized in ensureSSE after it receives a successful response to the initialize request
		// to avoid a race with marking the client as initialized here and sending the notifications/initialized message.
		return nil
	}

	sessionID := resp.Header.Get(SessionIDHeader)

	s.initializeLock.Lock()
	s.sessionID = &sessionID
	s.initializeRequest = &msg
	s.initializeLock.Unlock()

	go func() {
		if err = s.ensureSSE(ctx, nil, nil); err != nil {
			log.Errorf(context.Background(), "failed to initialize SSE: %v", err)
		}
	}()

	seen, err := s.readResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to decode mcp initialize response: %w", err)
	} else if !seen {
		return fmt.Errorf("no response from server, expected an initialize response")
	}

	return nil
}

func (s *HTTPClient) Send(ctx context.Context, msg Message) error {
	err := s.send(ctx, msg)
	if err == nil {
		return nil
	}

	// We need to check for various errors and handle them according the spec.

	// Check for an authentication-required error and put the user through the OAuth process.
	var oauthErr AuthRequiredErr
	if errors.As(err, &oauthErr) {
		httpClient, err := s.oauthHandler.oauthClient(s.ctx, s, s.baseURL, oauthErr.ProtectedResourceValue)
		if err != nil || httpClient == nil {
			streamError := fmt.Errorf("failed to initialize HTTP Streaming client: %w", oauthErr)
			return errors.Join(streamError, err)
		}

		s.clientLock.Lock()
		s.httpClient = httpClient
		s.clientLock.Unlock()

		// Make the call to send instead of Send so we don't get stuck in an authentication loop.
		return s.send(ctx, msg)
	}

	// Check for a session-not-found error and re-initialize.
	var sessionNotFoundErr SessionNotFoundErr
	if errors.As(err, &sessionNotFoundErr) && sessionNotFoundErr.SessionID != "" {
		s.initializeLock.Lock()
		s.sessionID = nil
		s.initializeLock.Unlock()

		// Make the call to send instead of Send so we don't get stuck in a reinitialize loop.
		return s.send(ctx, msg)
	}

	// This loop checks for errors from the oauth2 package we use for the HTTP client after authentication.
	// This is meant to catch errors such as failing to refresh the OAuth token.
	unwrappedErr := err
	for unwrappedErr != nil {
		// Continually unwrap the errors until we find one that starts with oauth2:
		if strings.HasPrefix(unwrappedErr.Error(), "oauth2:") {
			// If we do find an error that starts with "oauth2:" then there was an issue with the oauth2 HTTP client.
			// Reset the HTTP client to the default and try again. Using the default client will give us the unauthenticated
			// error that we need to continue the process.

			s.clientLock.Lock()
			s.httpClient = http.DefaultClient
			s.clientLock.Unlock()

			// Use the exported Send method here so that we catch the AuthRequiredErr above on the recursed call.
			return s.Send(ctx, msg)
		}
		unwrappedErr = errors.Unwrap(unwrappedErr)
	}

	return err
}

func (s *HTTPClient) send(ctx context.Context, msg Message) error {
	s.initializeLock.RLock()
	initialized := s.sessionID != nil
	initializeMessage := s.initializeRequest
	s.initializeLock.RUnlock()

	if !initialized {
		if msg.Method != "initialize" && initializeMessage == nil {
			return fmt.Errorf("cannot send %s message because client is not initialized, must send InitializeRequest first", msg.Method)
		}

		if initializeMessage == nil {
			initializeMessage = &msg
		} else {
			initializeMessage.ID = uuid.String()
		}
		if err := s.initialize(ctx, *initializeMessage); err != nil {
			return fmt.Errorf("failed to initialize client: %w", err)
		}

		if msg.Method == "initialize" {
			// If we're sending the request to initialize, then we're done.
			// Otherwise, we're reinitializing and should continue.
			return nil
		} else if err := s.send(ctx, Message{
			JSONRPC: "2.0",
			Method:  "notifications/initialized",
		}); err != nil {
			return fmt.Errorf("failed to send notifications/initialized: %w", err)
		}
	}

	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		// Ensure that the SSE connection is still active.
		if err := s.ensureSSE(ctx, initializeMessage, nil); err != nil {
			errChan <- fmt.Errorf("failed to restart SSE: %w", err)
		}
	}()

	if s.sse {
		// If this is an SSE-based MCP server, then we have to wait for the SSE connection to be established.
		if err := <-errChan; err != nil {
			return err
		}
	} else {
		// If not, then keep going. It will reconnect, if necessary.
		go func() {
			if err := <-errChan; err != nil {
				log.Errorf(ctx, "failed to restart SSE: %v", err)
			}
		}()
	}

	req, err := s.newRequest(ctx, http.MethodPost, msg)
	if err != nil {
		return err
	}

	s.clientLock.RLock()
	httpClient := s.httpClient
	s.clientLock.RUnlock()

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		streamingErrorMessage, _ := io.ReadAll(resp.Body)
		return AuthRequiredErr{
			ProtectedResourceValue: resp.Header.Get("WWW-Authenticate"),
			Err:                    fmt.Errorf("failed to send message: %s: %s", resp.Status, streamingErrorMessage),
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		streamingErrorMessage, _ := io.ReadAll(resp.Body)
		return SessionNotFoundErr{
			SessionID: req.Header.Get(SessionIDHeader),
			Err:       fmt.Errorf("failed to send message: %s: %s", resp.Status, streamingErrorMessage),
		}
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	if s.sse || resp.StatusCode == http.StatusAccepted {
		return nil
	}

	_, err = s.readResponse(resp)
	return err
}

func (s *HTTPClient) readResponse(resp *http.Response) (bool, error) {
	var seen bool
	handle := func(message *Message) {
		seen = true
		log.Messages(s.ctx, s.serverName, false, message.Result)
		go s.handler(s.ctx, *message)
	}

	if resp.Header.Get("Content-Type") == "text/event-stream" {
		stream := newSSEStream(resp.Body)
		for {
			data, ok := stream.readNextMessage()
			if !ok {
				return seen, nil
			}

			var message Message
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				return seen, fmt.Errorf("failed to decode response: %w", err)
			}

			handle(&message)
		}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return seen, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(data) == 0 {
		return false, nil
	}

	if data[0] != '{' {
		return false, fmt.Errorf("invalid response format, expected JSON object, got: %s", data)
	}

	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	handle(&message)
	return seen, nil
}

type SSEStream struct {
	lines *bufio.Scanner
}

func newSSEStream(input io.Reader) *SSEStream {
	lines := bufio.NewScanner(input)
	lines.Buffer(make([]byte, 0, 1024), 10*1024*1024)
	return &SSEStream{
		lines: lines,
	}
}

func (s *SSEStream) err() error {
	return s.lines.Err()
}

func (s *SSEStream) readNextMessage() (string, bool) {
	var eventName string
	for s.lines.Scan() {
		line := s.lines.Text()
		if len(line) == 0 {
			eventName = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") && (eventName == "message" || eventName == "" || eventName == "endpoint") {
			return strings.TrimSpace(line[5:]), true
		}
	}

	return "", false
}
