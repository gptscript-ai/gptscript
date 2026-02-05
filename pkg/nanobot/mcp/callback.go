package mcp

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

type AuthURLHandler interface {
	HandleAuthURL(context.Context, string, string) (bool, error)
}

type CallbackHandler interface {
	AuthURLHandler
	NewState(context.Context, *oauth2.Config, string) (string, <-chan CallbackPayload, error)
}

type CallbackServer interface {
	http.Handler
	CallbackHandler
}

type CallbackPayload struct {
	Code             string `json:"code"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type callbackHandler struct {
	AuthURLHandler
	lock  *sync.Mutex
	state map[string]callback
}

func NewCallbackServer(authURLHandler AuthURLHandler) CallbackServer {
	return &callbackHandler{
		lock:           new(sync.Mutex),
		state:          make(map[string]callback),
		AuthURLHandler: authURLHandler,
	}
}

func (s *callbackHandler) NewState(_ context.Context, conf *oauth2.Config, _ string) (string, <-chan CallbackPayload, error) {
	state := strings.ToLower(rand.Text())
	ch := make(chan CallbackPayload)
	s.lock.Lock()
	s.state[state] = callback{
		conf: conf,
		ch:   ch,
	}
	s.lock.Unlock()
	return state, ch, nil
}

func (s *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")

	s.lock.Lock()
	c, ok := s.state[state]
	delete(s.state, state)
	s.lock.Unlock()

	if !ok {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	c.ch <- CallbackPayload{
		Code:             r.URL.Query().Get("code"),
		Error:            r.URL.Query().Get("error"),
		ErrorDescription: r.URL.Query().Get("error_description"),
	}
	close(c.ch)

	_, _ = w.Write([]byte("Success!!"))
}

type callback struct {
	conf *oauth2.Config
	ch   chan<- CallbackPayload
}
