package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/complete"
	"github.com/gptscript-ai/gptscript/pkg/nanobot/uuid"
)

var ErrNoResult = errors.New("no result in response")

type MessageHandler interface {
	OnMessage(ctx context.Context, msg Message)
}

type MessageHandlerFunc func(ctx context.Context, msg Message)

func (f MessageHandlerFunc) OnMessage(ctx context.Context, msg Message) {
	f(ctx, msg)
}

type Wire interface {
	Close(deleteSession bool)
	Wait()
	Start(ctx context.Context, handler WireHandler) error
	Send(ctx context.Context, req Message) error
	SessionID() string
}

type WireHandler func(ctx context.Context, msg Message)

var sessionKey = struct{}{}

func SessionFromContext(ctx context.Context) *Session {
	if ctx == nil {
		return nil
	}
	s, ok := ctx.Value(sessionKey).(*Session)
	if !ok {
		return nil
	}
	return s
}

func WithSession(ctx context.Context, s *Session) context.Context {
	if s == nil {
		return ctx
	}
	return context.WithValue(ctx, sessionKey, s)
}

type Session struct {
	ctx               context.Context
	cancel            context.CancelFunc
	wire              Wire
	handler           MessageHandler
	pendingRequest    PendingRequests
	InitializeResult  InitializeResult
	InitializeRequest InitializeRequest
	recorder          *recorder
	Parent            *Session
	attributes        map[string]any
	lock              sync.Mutex
}

const SessionEnvMapKey = "env"

func (s *Session) Context() context.Context {
	return s.ctx
}

func (s *Session) ID() string {
	if s == nil || s.wire == nil {
		return ""
	}
	return s.wire.SessionID()
}

func (s *Session) State() (*SessionState, error) {
	if s == nil {
		return nil, nil
	}

	attr := make(map[string]any, len(s.attributes))
	for k, v := range s.attributes {
		if serializable, ok := v.(Serializable); ok {
			data, err := serializable.Serialize()
			if err != nil {
				return nil, fmt.Errorf("failed to serialize attribute %s: %w", k, err)
			}
			if data != nil {
				attr[k] = data
			}
		}
	}

	return &SessionState{
		ID:                s.wire.SessionID(),
		InitializeResult:  s.InitializeResult,
		InitializeRequest: s.InitializeRequest,
		Attributes:        attr,
	}, nil
}

func (s *Session) EnvMap() map[string]string {
	if s == nil {
		return map[string]string{}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.attributes == nil {
		s.attributes = make(map[string]any)
	}

	env, ok := s.attributes[SessionEnvMapKey].(map[string]string)
	if !ok {
		env = make(map[string]string)
		s.attributes[SessionEnvMapKey] = env
	}

	if s.Parent != nil {
		parentEnv := s.Parent.EnvMap()
		for k, v := range parentEnv {
			if _, exists := env[k]; !exists {
				env[k] = v
			}
		}
	}

	return env
}

func (s *Session) Delete(key string) {
	if s == nil {
		return
	}
	if s.Parent != nil {
		defer s.Parent.Delete(key)
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.attributes, key)
}

func (s *Session) Set(key string, value any) {
	if s == nil {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.attributes == nil {
		s.attributes = make(map[string]any)
	}
	s.attributes[key] = value
}

func (s *Session) copyInto(out, in any) bool {
	dstVal := reflect.ValueOf(out)
	srcVal := reflect.ValueOf(in)
	if srcVal.Type().AssignableTo(dstVal.Type()) {
		reflect.Indirect(dstVal).Set(reflect.Indirect(srcVal))
		return true
	}

	if dstVal.Type().Kind() == reflect.Ptr && srcVal.Type().AssignableTo(dstVal.Type().Elem()) {
		dstVal.Elem().Set(srcVal)
		return true
	}

	switch v := in.(type) {
	case float64:
		if outNum, ok := out.(*float64); ok {
			*outNum = v
			return true
		}
	case SavedString:
		if outStr, ok := out.(*string); ok {
			*outStr = string(v)
			return true
		}
	case string:
		if outStr, ok := out.(*string); ok {
			*outStr = v
			return true
		}
	}

	return false
}

func (s *Session) Get(key string, out any) (ret bool) {
	if s == nil {
		return false
	}
	defer func() {
		if !ret && s != nil && s.Parent != nil {
			ret = s.Parent.Get(key, out)
		}
	}()

	s.lock.Lock()
	v, ok := s.attributes[key]
	if !ok {
		s.lock.Unlock()
		return false
	}
	s.lock.Unlock()

	if v == nil {
		return false
	}

	if out == nil {
		return true
	}

	if s.copyInto(out, v) {
		return true
	}

	if deserializable, ok := out.(Deserializable); ok {
		newOut, err := deserializable.Deserialize(v)
		if err != nil {
			s.lock.Lock()
			delete(s.attributes, key)
			s.lock.Unlock()
			return false
		}
		s.lock.Lock()
		s.attributes[key] = newOut
		s.lock.Unlock()
		return true
	}

	panic(fmt.Sprintf("can not marshal %T to type: %T", v, out))
}

func (s *Session) Attributes() map[string]any {
	if s == nil || len(s.attributes) == 0 {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	return maps.Clone(s.attributes)
}

func (s *Session) Close(deleteSession bool) {
	if s.wire != nil {
		s.wire.Close(deleteSession)
	}
	s.pendingRequest.Close()
	s.cancel()
}

func (s *Session) Wait() {
	if s.wire == nil {
		<-s.ctx.Done()
		return
	}
	s.wire.Wait()
}

func (s *Session) normalizeProgress(progress *NotificationProgressRequest) {
	var (
		progressKey               = fmt.Sprintf("progress-token:%v", progress.ProgressToken)
		lastProgress, newProgress float64
	)

	if ok := s.Get(progressKey, &lastProgress); !ok {
		lastProgress = 0
	}

	if progress.Progress != "" {
		newF, err := progress.Progress.Float64()
		if err == nil {
			newProgress = newF
		}
	}

	if newProgress <= lastProgress {
		if progress.Total == nil {
			newProgress = lastProgress + 1
		} else {
			// If total is set then something is probably trying to make the progress pretty
			// so we don't want to just increment by 1 and mess that up.
			newProgress = lastProgress + 0.01
		}
	}
	data, err := json.Marshal(newProgress)
	if err == nil {
		progress.Progress = json.Number(data)
	}
	s.Set(progressKey, newProgress)
}

func (s *Session) SendPayload(ctx context.Context, method string, payload any) error {
	if progress, ok := payload.(NotificationProgressRequest); ok {
		s.normalizeProgress(&progress)
		payload = progress
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return s.Send(ctx, Message{
		Method: method,
		Params: data,
	})
}

func (s *Session) Send(ctx context.Context, req Message) error {
	if s.wire == nil {
		return fmt.Errorf("empty session: wire is not initialized")
	}

	req.JSONRPC = "2.0"
	s.recorder.save(ctx, s.wire.SessionID(), true, req)
	return s.wire.Send(ctx, req)
}

type ExchangeOption struct {
	ProgressToken any
}

func (e ExchangeOption) Merge(other ExchangeOption) (result ExchangeOption) {
	result.ProgressToken = complete.Last(e.ProgressToken, other.ProgressToken)
	return
}

func (s *Session) preInit(msg *Message) (bool, error) {
	if msg.Method == "initialize" {
		var init InitializeRequest
		if err := json.Unmarshal(msg.Params, &init); err != nil {
			return false, fmt.Errorf("failed to unmarshal initialize request: %w", err)
		}
		s.InitializeRequest = init
		return true, nil
	}

	return false, nil
}

func (s *Session) postInit(msg *Message) error {
	if len(msg.Result) == 0 {
		return nil
	}
	var init InitializeResult
	if err := json.Unmarshal(msg.Result, &init); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}
	s.InitializeResult = init
	return nil
}

func (s *Session) marshalResponse(m Message, out any) error {
	if mOut, ok := out.(*Message); ok {
		*mOut = m
		return nil
	}
	if m.Error != nil {
		return fmt.Errorf("error from server: %s", m.Error.Message)
	}
	if m.Result == nil {
		return ErrNoResult
	}
	if err := json.Unmarshal(m.Result, out); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return nil
}

func (s *Session) toRequest(method string, in any, opt ExchangeOption) (*Message, error) {
	req, ok := in.(*Message)
	if !ok {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		req = &Message{
			Method: method,
			Params: data,
		}
	}

	if req.ID == nil || req.ID == "" {
		req.ID = uuid.String()
	}
	if opt.ProgressToken != nil {
		if err := req.SetProgressToken(opt.ProgressToken); err != nil {
			return nil, fmt.Errorf("failed to set progress token: %w", err)
		}
	}

	return req, nil
}

func (s *Session) Exchange(ctx context.Context, method string, in, out any, opts ...ExchangeOption) error {
	opt := complete.Complete(opts...)
	req, err := s.toRequest(method, in, opt)
	if err != nil {
		return err
	}

	if req.ID == nil {
		return s.Send(ctx, *req)
	}

	ch := s.pendingRequest.WaitFor(req.ID)
	defer s.pendingRequest.Done(req.ID)

	isInit, err := s.preInit(req)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		if err := s.Send(ctx, *req); err != nil {
			errChan <- fmt.Errorf("failed to send request: %w", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err = <-errChan:
			if err != nil {
				return err
			}
			// If the error is nil, then the send call was successful.
			// Set the error channel to nil so that this case always blocks.
			errChan = nil
		case m := <-ch:
			if isInit {
				if err := s.postInit(&m); err != nil {
					return fmt.Errorf("failed to post init: %w", err)
				}
			}
			return s.marshalResponse(m, out)
		}
	}
}

func (s *Session) onWire(ctx context.Context, message Message) {
	s.recorder.save(s.ctx, s.wire.SessionID(), false, message)
	message.Session = s
	if s.pendingRequest.Notify(message) {
		return
	}
	s.handler.OnMessage(WithSession(ctx, s), message)
}

func NewEmptySession(ctx context.Context) *Session {
	s := &Session{}
	s.ctx, s.cancel = context.WithCancel(WithSession(ctx, s))
	return s
}

func newSession(ctx context.Context, wire Wire, handler MessageHandler, session *SessionState, r *recorder) (*Session, error) {
	s := &Session{
		wire:     wire,
		handler:  handler,
		recorder: r,
	}
	if session != nil {
		s.InitializeRequest = session.InitializeRequest
		s.InitializeResult = session.InitializeResult
	}
	withSession := WithSession(ctx, s)
	s.ctx, s.cancel = context.WithCancel(withSession)

	if err := wire.Start(s.ctx, s.onWire); err != nil {
		return nil, err
	}

	go func() {
		wire.Wait()
		s.Close(false)
	}()

	return s, nil
}

type recorder struct {
}

func (r *recorder) save(ctx context.Context, sessionID string, send bool, msg Message) {
}

type Serializable interface {
	Serialize() (any, error)
}

type Deserializable interface {
	Deserialize(v any) (any, error)
}

type Closer interface {
	Close() error
}

type SavedString string

func (s SavedString) Serialize() (any, error) {
	return s, nil
}
