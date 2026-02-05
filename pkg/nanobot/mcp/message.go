package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/nanobot/log"
)

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`

	Session *Session `json:"-"`
}

func NewMessage(method string, params any) (*Message, error) {
	msg := &Message{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		msg.Params = data
	}
	return msg, nil
}

func (r *Message) IsRequest() bool {
	return len(r.Params) > 0 && !bytes.Equal(r.Params, []byte("null"))
}

func (r *Message) SetProgressToken(token any) error {
	params := map[string]any{}
	if len(r.Params) > 0 {
		if err := json.Unmarshal(r.Params, &params); err != nil {
			return fmt.Errorf("failed to unmarshal params to set progress token: %w", err)
		}
	}

	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		meta = make(map[string]any)
	}

	meta["progressToken"] = token
	params["_meta"] = meta
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params to set progress token: %w", err)
	}

	r.Params = data
	return nil
}

func (r *Message) ProgressToken() any {
	if len(r.Params) == 0 || !bytes.Contains(r.Params, []byte("progressToken")) {
		return nil
	}
	var token struct {
		Meta struct {
			ProgressToken any `json:"progressToken"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(r.Params, &token); err == nil && token.Meta.ProgressToken != nil {
		return token.Meta.ProgressToken
	}
	return nil
}

func (r *Message) UID(sessionID string, in bool) string {
	parts := strings.Split(sessionID, "/")
	sessionID, _, _ = strings.Cut(parts[len(parts)-1], "::")

	var (
		id        = fmt.Sprintf("%v", r.ID)
		direction = "out"
	)
	if in {
		direction = "in"
	}
	return fmt.Sprintf("%s::%s::%s", sessionID, id, direction)
}

func (r *Message) SendError(ctx context.Context, err error) {
	if r.Session == nil {
		return
	}
	var data *RPCError
	if rpcError := (JSONRPCError)(nil); errors.As(err, &rpcError) {
		data = rpcError.RPCError()
	} else {
		data = ErrRPCInternal.WithMessage("%s", err.Error())
	}

	resp := Message{
		JSONRPC: r.JSONRPC,
		ID:      r.ID,
		Error:   data,
	}

	if err := r.Session.Send(ctx, resp); err != nil {
		log.Errorf(ctx, "failed to send error response: %v", err)
	}
}

func (r *Message) Reply(ctx context.Context, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	return r.Session.Send(ctx, Message{
		JSONRPC: r.JSONRPC,
		ID:      r.ID,
		Result:  data,
	})
}

type JSONRPCError interface {
	RPCError() *RPCError
}

var (
	ErrRPCParse          = NewRPCError(-32700, "JSON RPC parse error")
	ErrRPCInvalidRequest = NewRPCError(-32600, "JSON RPC invalid request")
	ErrRPCMethodNotFound = NewRPCError(-32601, "JSON RPC method not found")
	ErrRPCInvalidParams  = NewRPCError(-32602, "JSON RPC invalid params")
	ErrRPCInternal       = NewRPCError(-32603, "JSON RPC internal error")
)

type RPCError struct {
	Code       int             `json:"code,omitempty"`
	Message    string          `json:"message,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	DataObject any             `json:"-"`
}

func NewRPCError(code int, message string) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
	}
}

func (e *RPCError) WithMessage(fmtStr string, args ...any) *RPCError {
	cp := *e
	cp.Message += ": " + fmt.Sprintf(fmtStr, args...)
	return &cp
}

func (e *RPCError) RPCError() *RPCError {
	if e == nil {
		return nil
	}
	if e.DataObject != nil {
		result := *e
		result.Data, _ = json.Marshal(e.DataObject)
		return &result
	}
	return e
}

func (e *RPCError) Error() string {
	if e == nil {
		return "nil error"
	}
	if e.Data != nil {
		return fmt.Sprintf("%d: %s (%s)", e.Code, e.Message, string(e.Data))
	}
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}
