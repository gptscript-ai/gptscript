package mcp

import (
	"context"
	"errors"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/uuid"
)

var (
	_ Wire = (*serverWire)(nil)
	_ Wire = (*ServerSession)(nil)
)

func NewServerSession(ctx context.Context, handler MessageHandler) (*ServerSession, error) {
	return NewExistingServerSession(ctx,
		SessionState{
			ID: uuid.String(),
		}, handler)
}

func NewExistingServerSession(ctx context.Context, state SessionState, handler MessageHandler) (*ServerSession, error) {
	s := &serverWire{
		read:      make(chan Message),
		noReader:  make(chan struct{}),
		sessionID: state.ID,
	}
	s.stopReading()

	session, err := newSession(ctx, s, handler, &state, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range state.Attributes {
		session.Set(k, v)
	}
	session.Parent = SessionFromContext(ctx)
	return &ServerSession{
		session: session,
		wire:    s,
	}, nil
}

type ServerSession struct {
	session *Session
	wire    *serverWire
}

func (s *ServerSession) Wait() {
	if s.session == nil {
		return
	}
	s.session.Wait()
}

func (s *ServerSession) Start(ctx context.Context, handler WireHandler) error {
	s.wire.startReading()

	go func() {
		defer s.wire.stopReading()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-s.wire.read:
				if !ok {
					return
				}
				handler(ctx, msg)
			}
		}
	}()
	return nil
}

func (s *ServerSession) SessionID() string {
	return s.ID()
}

func (s *ServerSession) ID() string {
	id := s.session.ID()
	if id == "" {
		return s.wire.SessionID()
	}
	return id
}

var (
	ErrNoResponse = errors.New("no response")
	ErrNoReader   = errors.New("no reader")
)

func (s *ServerSession) GetSession() *Session {
	return s.session
}

func (s *ServerSession) Exchange(ctx context.Context, msg Message) (Message, error) {
	isInit, err := s.session.preInit(&msg)
	if err != nil {
		return Message{}, err
	}
	resp, err := s.wire.exchange(ctx, msg)
	if err != nil {
		return Message{}, err
	}
	if isInit {
		if err := s.session.postInit(&resp); err != nil {
			return Message{}, err
		}
	}
	return resp, nil
}

func (s *ServerSession) Read(ctx context.Context) (Message, bool) {
	select {
	case msg, ok := <-s.wire.read:
		if !ok {
			return Message{}, false
		}
		return msg, true
	case <-ctx.Done():
		return Message{}, false
	}
}

func (s *ServerSession) StartReading() {
	s.wire.startReading()
}

func (s *ServerSession) StopReading() {
	s.wire.stopReading()
}

func (s *ServerSession) Send(ctx context.Context, req Message) error {
	req.Session = s.session
	go s.session.handler.OnMessage(WithSession(ctx, s.session), req)
	return nil
}

func (s *ServerSession) Close(deleteSession bool) {
	if s == nil {
		return
	}

	if s.session == nil {
		s.session.Close(deleteSession)
	}
	if s.wire != nil {
		s.wire.Close(deleteSession)
	}
}

type serverWire struct {
	ctx        context.Context
	cancel     context.CancelFunc
	pending    PendingRequests
	read       chan Message
	readerLock sync.RWMutex
	noReader   chan struct{}
	handler    WireHandler
	sessionID  string
}

func (s *serverWire) SessionID() string {
	return s.sessionID
}

func (s *serverWire) exchange(ctx context.Context, msg Message) (Message, error) {
	if msg.ID == nil {
		s.handler(ctx, msg)
		return Message{}, ErrNoResponse
	}

	ch := s.pending.WaitFor(msg.ID)
	defer s.pending.Done(msg.ID)

	go func() {
		s.handler(ctx, msg)
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return Message{}, ctx.Err()
	case <-s.ctx.Done():
		return Message{}, s.ctx.Err()
	case m, ok := <-ch:
		if !ok {
			return Message{}, ErrNoResponse
		}
		return m, nil
	}
}

func (s *serverWire) Close(bool) {
	s.cancel()
}

func (s *serverWire) Wait() {
	<-s.ctx.Done()
}

func (s *serverWire) Start(ctx context.Context, handler WireHandler) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.handler = handler
	return nil
}

func (s *serverWire) Send(ctx context.Context, req Message) error {
	if s.pending.Notify(req) {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.noReader:
		return ErrNoReader
	case s.read <- req:
		return nil
	}
}

func (s *serverWire) startReading() {
	s.readerLock.Lock()
	defer s.readerLock.Unlock()

	s.noReader = nil
}

func (s *serverWire) stopReading() {
	s.readerLock.Lock()
	defer s.readerLock.Unlock()

	s.noReader = make(chan struct{})
	close(s.noReader)
}

func (s *serverWire) isReading() bool {
	s.readerLock.RLock()
	defer s.readerLock.RUnlock()

	return s.noReader != nil
}
