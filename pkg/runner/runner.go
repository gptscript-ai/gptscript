package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/config"
	context2 "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"golang.org/x/exp/maps"
)

type MonitorFactory interface {
	Start(ctx context.Context, prg *types.Program, env []string, input string) (Monitor, error)
}

type Monitor interface {
	Event(event Event)
	Pause() func()
	Stop(output string, err error)
}

type Options struct {
	MonitorFactory     MonitorFactory        `usage:"-"`
	RuntimeManager     engine.RuntimeManager `usage:"-"`
	StartPort          int64                 `usage:"-"`
	EndPort            int64                 `usage:"-"`
	CredentialOverride string                `usage:"-"`
	Sequential         bool                  `usage:"-"`
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.MonitorFactory = types.FirstSet(opt.MonitorFactory, result.MonitorFactory)
		result.RuntimeManager = types.FirstSet(opt.RuntimeManager, result.RuntimeManager)
		result.StartPort = types.FirstSet(opt.StartPort, result.StartPort)
		result.EndPort = types.FirstSet(opt.EndPort, result.EndPort)
		result.CredentialOverride = types.FirstSet(opt.CredentialOverride, result.CredentialOverride)
		result.Sequential = types.FirstSet(opt.Sequential, result.Sequential)
	}
	if result.MonitorFactory == nil {
		result.MonitorFactory = noopFactory{}
	}
	if result.EndPort == 0 {
		result.EndPort = result.StartPort
	}
	if result.StartPort == 0 {
		result.StartPort = result.EndPort
	}
	return
}

type Runner struct {
	c              engine.Model
	factory        MonitorFactory
	runtimeManager engine.RuntimeManager
	ports          engine.Ports
	credCtx        string
	credMutex      sync.Mutex
	credOverrides  string
	sequential     bool
}

func New(client engine.Model, credCtx string, opts ...Options) (*Runner, error) {
	opt := complete(opts...)

	runner := &Runner{
		c:              client,
		factory:        opt.MonitorFactory,
		runtimeManager: opt.RuntimeManager,
		credCtx:        credCtx,
		credMutex:      sync.Mutex{},
		credOverrides:  opt.CredentialOverride,
		sequential:     opt.Sequential,
	}

	if opt.StartPort != 0 {
		if opt.EndPort < opt.StartPort {
			return nil, fmt.Errorf("invalid port range: %d-%d", opt.StartPort, opt.EndPort)
		}
		runner.ports.SetPorts(opt.StartPort, opt.EndPort)
	}

	return runner, nil
}

func (r *Runner) Close() {
	r.ports.CloseDaemons()
}

type ChatResponse struct {
	Done    bool      `json:"done"`
	Content string    `json:"content"`
	ToolID  string    `json:"toolID"`
	State   ChatState `json:"state"`
}

type ChatState interface{}

func (r *Runner) Chat(ctx context.Context, prevState ChatState, prg types.Program, env []string, input string) (resp ChatResponse, err error) {
	var state *State

	if prevState != nil {
		switch v := prevState.(type) {
		case *State:
			state = v
		case string:
			if v != "null" {
				state = &State{}
				if err := json.Unmarshal([]byte(v), state); err != nil {
					return resp, fmt.Errorf("failed to unmarshal chat state: %w", err)
				}
			}
		default:
			return resp, fmt.Errorf("invalid type for state object: %T", prevState)
		}
	}

	monitor, err := r.factory.Start(ctx, &prg, env, input)
	if err != nil {
		return resp, err
	}
	defer func() {
		monitor.Stop(resp.Content, err)
	}()

	callCtx := engine.NewContext(ctx, &prg)
	if state == nil {
		startResult, err := r.start(callCtx, monitor, env, input)
		if err != nil {
			return resp, err
		}
		state = &State{
			Continuation: startResult,
		}
	} else {
		state.ResumeInput = &input
	}

	state, err = r.resume(callCtx, monitor, env, state)
	if err != nil {
		return resp, err
	}

	if state.Result != nil {
		return ChatResponse{
			Done:    true,
			Content: *state.Result,
		}, nil
	}

	content, err := state.ContinuationContent()
	if err != nil {
		return resp, err
	}

	toolID, err := state.ContinuationContentToolID()
	if err != nil {
		return resp, err
	}

	return ChatResponse{
		Content: content,
		State:   state,
		ToolID:  toolID,
	}, nil
}

func (r *Runner) Run(ctx context.Context, prg types.Program, env []string, input string) (output string, err error) {
	resp, err := r.Chat(ctx, nil, prg, env, input)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

type Event struct {
	Time               time.Time              `json:"time,omitempty"`
	CallContext        *engine.CallContext    `json:"callContext,omitempty"`
	ToolSubCalls       map[string]engine.Call `json:"toolSubCalls,omitempty"`
	ToolResults        int                    `json:"toolResults,omitempty"`
	Type               EventType              `json:"type,omitempty"`
	ChatCompletionID   string                 `json:"chatCompletionId,omitempty"`
	ChatRequest        any                    `json:"chatRequest,omitempty"`
	ChatResponse       any                    `json:"chatResponse,omitempty"`
	ChatResponseCached bool                   `json:"chatResponseCached,omitempty"`
	Content            string                 `json:"content,omitempty"`
}

type EventType string

var (
	EventTypeCallStart    = EventType("callStart")
	EventTypeCallContinue = EventType("callContinue")
	EventTypeCallSubCalls = EventType("callSubCalls")
	EventTypeCallProgress = EventType("callProgress")
	EventTypeChat         = EventType("callChat")
	EventTypeCallFinish   = EventType("callFinish")
)

func getContextInput(prg *types.Program, ref types.ToolReference, input string) (string, error) {
	if ref.Arg == "" {
		return "", nil
	}

	targetArgs := prg.ToolSet[ref.ToolID].Arguments
	targetKeys := map[string]string{}

	if targetArgs == nil {
		return "", nil
	}

	for targetKey := range targetArgs.Properties {
		targetKeys[strings.ToLower(targetKey)] = targetKey
	}

	inputMap := map[string]interface{}{}
	outputMap := map[string]interface{}{}

	_ = json.Unmarshal([]byte(input), &inputMap)

	fields := strings.Fields(ref.Arg)

	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "and" {
			continue
		}
		if field == "as" {
			i++
			continue
		}

		var (
			keyName string
			val     any
		)

		if strings.HasPrefix(field, "$") {
			key := strings.TrimPrefix(field, "$")
			key = strings.TrimPrefix(key, "{")
			key = strings.TrimSuffix(key, "}")
			val = inputMap[key]
		} else {
			val = field
		}

		if len(fields) > i+1 && fields[i+1] == "as" {
			keyName = strings.ToLower(fields[i+2])
		}

		if len(targetKeys) == 0 {
			return "", fmt.Errorf("can not assign arg to context because target tool [%s] has no defined args", ref.ToolID)
		}

		if keyName == "" {
			if len(targetKeys) != 1 {
				return "", fmt.Errorf("can not assign arg to context because target tool [%s] has does not have one args. You must use \"as\" syntax to map the arg to a key %v", ref.ToolID, targetKeys)
			}
			for k := range targetKeys {
				keyName = k
			}
		}

		if targetKey, ok := targetKeys[strings.ToLower(keyName)]; ok {
			outputMap[targetKey] = val
		} else {
			return "", fmt.Errorf("can not assign arg to context because target tool [%s] has does not args [%s]", ref.ToolID, keyName)
		}
	}

	if len(outputMap) == 0 {
		return "", nil
	}

	output, err := json.Marshal(outputMap)
	return string(output), err
}

func (r *Runner) getContext(callCtx engine.Context, monitor Monitor, env []string, input string) (result []engine.InputContext, _ error) {
	toolRefs, err := callCtx.Program.GetContextToolRefs(callCtx.Tool.ID)
	if err != nil {
		return nil, err
	}

	for _, toolRef := range toolRefs {
		contextInput, err := getContextInput(callCtx.Program, toolRef, input)
		if err != nil {
			return nil, err
		}

		content, err := r.subCall(callCtx.Ctx, callCtx, monitor, env, toolRef.ToolID, contextInput, "", engine.ContextToolCategory)
		if err != nil {
			return nil, err
		}
		if content.Result == nil {
			return nil, fmt.Errorf("context tool can not result in a chat continuation")
		}
		result = append(result, engine.InputContext{
			ToolID:  toolRef.ToolID,
			Content: *content.Result,
		})
	}
	return result, nil
}

func (r *Runner) call(callCtx engine.Context, monitor Monitor, env []string, input string) (*State, error) {
	result, err := r.start(callCtx, monitor, env, input)
	if err != nil {
		return nil, err
	}
	return r.resume(callCtx, monitor, env, &State{
		Continuation: result,
	})
}

func (r *Runner) start(callCtx engine.Context, monitor Monitor, env []string, input string) (*engine.Return, error) {
	progress, progressClose := streamProgress(&callCtx, monitor)
	defer progressClose()

	if len(callCtx.Tool.Credentials) > 0 {
		var err error
		env, err = r.handleCredentials(callCtx, monitor, env)
		if err != nil {
			return nil, err
		}
	}

	var err error
	callCtx.InputContext, err = r.getContext(callCtx, monitor, env, input)
	if err != nil {
		return nil, err
	}

	e := engine.Engine{
		Model:          r.c,
		RuntimeManager: r.runtimeManager,
		Progress:       progress,
		Env:            env,
		Ports:          &r.ports,
	}

	monitor.Event(Event{
		Time:        time.Now(),
		CallContext: callCtx.GetCallContext(),
		Type:        EventTypeCallStart,
		Content:     input,
	})

	callCtx.Ctx = context2.AddPauseFuncToCtx(callCtx.Ctx, monitor.Pause)

	return e.Start(callCtx, input)
}

type State struct {
	Continuation       *engine.Return `json:"continuation,omitempty"`
	ContinuationToolID string         `json:"continuationToolID,omitempty"`
	Result             *string        `json:"result,omitempty"`

	ResumeInput *string         `json:"resumeInput,omitempty"`
	SubCalls    []SubCallResult `json:"subCalls,omitempty"`
	SubCallID   string          `json:"subCallID,omitempty"`
}

func (s State) WithInput(input *string) *State {
	s.ResumeInput = input
	return &s
}

func (s State) ContinuationContentToolID() (string, error) {
	if s.Continuation.Result != nil {
		return s.ContinuationToolID, nil
	}

	for _, subCall := range s.SubCalls {
		if s.SubCallID == subCall.CallID {
			return subCall.State.ContinuationContentToolID()
		}
	}
	return "", fmt.Errorf("illegal state: no result message found in chat response")
}

func (s State) ContinuationContent() (string, error) {
	if s.Continuation.Result != nil {
		return *s.Continuation.Result, nil
	}

	for _, subCall := range s.SubCalls {
		if s.SubCallID == subCall.CallID {
			return subCall.State.ContinuationContent()
		}
	}
	return "", fmt.Errorf("illegal state: no result message found in chat response")
}

type Needed struct {
	Content string `json:"content,omitempty"`
	Input   string `json:"input,omitempty"`
}

func (r *Runner) resume(callCtx engine.Context, monitor Monitor, env []string, state *State) (*State, error) {
	progress, progressClose := streamProgress(&callCtx, monitor)
	defer progressClose()

	if len(callCtx.Tool.Credentials) > 0 {
		var err error
		env, err = r.handleCredentials(callCtx, monitor, env)
		if err != nil {
			return nil, err
		}
	}

	var (
		err          error
		contentInput string
	)

	if state.Continuation != nil && state.Continuation.State != nil {
		contentInput = state.Continuation.State.Input
	}

	callCtx.InputContext, err = r.getContext(callCtx, monitor, env, contentInput)
	if err != nil {
		return nil, err
	}

	e := engine.Engine{
		Model:          r.c,
		RuntimeManager: r.runtimeManager,
		Progress:       progress,
		Env:            env,
		Ports:          &r.ports,
	}

	for {
		if state.Continuation.Result != nil && len(state.Continuation.Calls) == 0 && state.SubCallID == "" && state.ResumeInput == nil {
			progressClose()
			monitor.Event(Event{
				Time:        time.Now(),
				CallContext: callCtx.GetCallContext(),
				Type:        EventTypeCallFinish,
				Content:     *state.Continuation.Result,
			})
			if callCtx.Tool.Chat {
				return &State{
					Continuation:       state.Continuation,
					ContinuationToolID: callCtx.Tool.ID,
				}, nil
			}
			return &State{
				Result: state.Continuation.Result,
			}, nil
		}

		monitor.Event(Event{
			Time:         time.Now(),
			CallContext:  callCtx.GetCallContext(),
			Type:         EventTypeCallSubCalls,
			ToolSubCalls: state.Continuation.Calls,
		})

		var (
			callResults []SubCallResult
			err         error
		)

		state, callResults, err = r.subCalls(callCtx, monitor, env, state)
		if errMessage := (*builtin.ErrChatFinish)(nil); errors.As(err, &errMessage) && callCtx.Tool.Chat {
			return &State{
				Result: &errMessage.Message,
			}, nil
		} else if err != nil {
			return nil, err
		}

		var engineResults []engine.CallResult
		for _, callResult := range callResults {
			if callResult.State.Continuation == nil {
				engineResults = append(engineResults, engine.CallResult{
					ToolID: callResult.ToolID,
					CallID: callResult.CallID,
					Result: *callResult.State.Result,
				})
			} else {
				return &State{
					Continuation: state.Continuation,
					SubCalls:     callResults,
					SubCallID:    callResult.CallID,
				}, nil
			}
		}

		if state.ResumeInput != nil {
			engineResults = append(engineResults, engine.CallResult{
				User: *state.ResumeInput,
			})
		}

		monitor.Event(Event{
			Time:        time.Now(),
			CallContext: callCtx.GetCallContext(),
			Type:        EventTypeCallContinue,
			ToolResults: len(callResults),
		})

		nextContinuation, err := e.Continue(callCtx, state.Continuation.State, engineResults...)
		if err != nil {
			return nil, err
		}

		state = &State{
			Continuation: nextContinuation,
			SubCalls:     callResults,
		}
	}
}

func streamProgress(callCtx *engine.Context, monitor Monitor) (chan<- types.CompletionStatus, func()) {
	progress := make(chan types.CompletionStatus)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range progress {
			if message := status.PartialResponse; message != nil {
				monitor.Event(Event{
					Time:             time.Now(),
					CallContext:      callCtx.GetCallContext(),
					Type:             EventTypeCallProgress,
					ChatCompletionID: status.CompletionID,
					Content:          message.String(),
				})
			} else {
				monitor.Event(Event{
					Time:               time.Now(),
					CallContext:        callCtx.GetCallContext(),
					Type:               EventTypeChat,
					ChatCompletionID:   status.CompletionID,
					ChatRequest:        status.Request,
					ChatResponse:       status.Response,
					ChatResponseCached: status.Cached,
				})
			}
		}
	}()

	var once sync.Once
	return progress, func() {
		once.Do(func() {
			close(progress)
			wg.Wait()
		})
	}
}

func (r *Runner) subCall(ctx context.Context, parentContext engine.Context, monitor Monitor, env []string, toolID, input, callID string, toolCategory engine.ToolCategory) (*State, error) {
	callCtx, err := parentContext.SubCall(ctx, toolID, callID, toolCategory)
	if err != nil {
		return nil, err
	}

	return r.call(callCtx, monitor, env, input)
}

func (r *Runner) subCallResume(ctx context.Context, parentContext engine.Context, monitor Monitor, env []string, toolID, callID string, state *State) (*State, error) {
	callCtx, err := parentContext.SubCall(ctx, toolID, callID, engine.NoCategory)
	if err != nil {
		return nil, err
	}

	return r.resume(callCtx, monitor, env, state)
}

type SubCallResult struct {
	ToolID string `json:"toolId,omitempty"`
	CallID string `json:"callId,omitempty"`
	State  *State `json:"state,omitempty"`
}

func (r *Runner) newDispatcher(ctx context.Context) dispatcher {
	if r.sequential {
		return newSerialDispatcher(ctx)
	}
	return newParallelDispatcher(ctx)
}

func (r *Runner) subCalls(callCtx engine.Context, monitor Monitor, env []string, state *State) (_ *State, callResults []SubCallResult, _ error) {
	var (
		resultLock sync.Mutex
	)

	if state.SubCallID != "" {
		if state.ResumeInput == nil {
			return nil, nil, fmt.Errorf("invalid state, input must be set for sub call continuation on callID [%s]", state.SubCallID)
		}
		var found bool
		for _, subCall := range state.SubCalls {
			if subCall.CallID == state.SubCallID {
				found = true
				subState := *subCall.State
				subState.ResumeInput = state.ResumeInput
				result, err := r.subCallResume(callCtx.Ctx, callCtx, monitor, env, subCall.ToolID, subCall.CallID, subCall.State.WithInput(state.ResumeInput))
				if err != nil {
					return nil, nil, err
				}
				callResults = append(callResults, SubCallResult{
					ToolID: subCall.ToolID,
					CallID: subCall.CallID,
					State:  result,
				})
				// Clear the input, we have already processed it
				state = state.WithInput(nil)
			} else {
				callResults = append(callResults, subCall)
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("invalid state, failed to find subCall for subCallID [%s]", state.SubCallID)
		}
		return state, callResults, nil
	}

	d := r.newDispatcher(callCtx.Ctx)

	// Sort the id so if sequential the results are predictable
	ids := maps.Keys(state.Continuation.Calls)
	sort.Strings(ids)

	for _, id := range ids {
		call := state.Continuation.Calls[id]
		d.Run(func(ctx context.Context) error {
			result, err := r.subCall(ctx, callCtx, monitor, env, call.ToolID, call.Input, id, "")
			if err != nil {
				return err
			}

			resultLock.Lock()
			defer resultLock.Unlock()
			callResults = append(callResults, SubCallResult{
				ToolID: call.ToolID,
				CallID: id,
				State:  result,
			})

			return nil
		})
	}

	if err := d.Wait(); err != nil {
		return nil, nil, err
	}

	return state, callResults, nil
}

func (r *Runner) handleCredentials(callCtx engine.Context, monitor Monitor, env []string) ([]string, error) {
	// Since credential tools (usually) prompt the user, we want to only run one at a time.
	r.credMutex.Lock()
	defer r.credMutex.Unlock()

	// Set up the credential store.
	c, err := config.ReadCLIConfig("")
	if err != nil {
		return nil, fmt.Errorf("failed to read CLI config: %w", err)
	}

	store, err := credentials.NewStore(c, r.credCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials store: %w", err)
	}

	// Parse the credential overrides from the command line argument, if there are any.
	var credOverrides map[string]map[string]string
	if r.credOverrides != "" {
		credOverrides, err = parseCredentialOverrides(r.credOverrides)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credential overrides: %w", err)
		}
	}

	for _, credToolName := range callCtx.Tool.Credentials {
		// Check whether the credential was overridden before we attempt to find it in the store or run the tool.
		if override, exists := credOverrides[credToolName]; exists {
			for k, v := range override {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			continue
		}

		var (
			cred   *credentials.Credential
			exists bool
			err    error
		)

		// Only try to look up the cred if the tool is on GitHub.
		if isGitHubTool(credToolName) {
			cred, exists, err = store.Get(credToolName)
			if err != nil {
				return nil, fmt.Errorf("failed to get credentials for tool %s: %w", credToolName, err)
			}
		}

		// If the credential doesn't already exist in the store, run the credential tool in order to get the value,
		// and save it in the store.
		if !exists {
			credToolID, ok := callCtx.Tool.ToolMapping[credToolName]
			if !ok {
				return nil, fmt.Errorf("failed to find ID for tool %s", credToolName)
			}

			subCtx, err := callCtx.SubCall(callCtx.Ctx, credToolID, "", engine.CredentialToolCategory) // leaving callID as "" will cause it to be set by the engine
			if err != nil {
				return nil, fmt.Errorf("failed to create subcall context for tool %s: %w", credToolName, err)
			}

			res, err := r.call(subCtx, monitor, env, "")
			if err != nil {
				return nil, fmt.Errorf("failed to run credential tool %s: %w", credToolName, err)
			}

			if res.Result == nil {
				return nil, fmt.Errorf("invalid state: credential tool [%s] can not result in a continuation", credToolName)
			}

			var envMap struct {
				Env map[string]string `json:"env"`
			}
			if err := json.Unmarshal([]byte(*res.Result), &envMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal credential tool %s response: %w", credToolName, err)
			}

			cred = &credentials.Credential{
				ToolName: credToolName,
				Env:      envMap.Env,
			}

			isEmpty := true
			for _, v := range cred.Env {
				if v != "" {
					isEmpty = false
					break
				}
			}

			// Only store the credential if the tool is on GitHub, and the credential is non-empty.
			if isGitHubTool(credToolName) && callCtx.Program.ToolSet[credToolID].Source.Repo != nil {
				if isEmpty {
					log.Warnf("Not saving empty credential for tool %s", credToolName)
				} else if err := store.Add(*cred); err != nil {
					return nil, fmt.Errorf("failed to add credential for tool %s: %w", credToolName, err)
				}
			} else {
				log.Warnf("Not saving credential for local tool %s - credentials will only be saved for tools from GitHub.", credToolName)
			}
		}

		for k, v := range cred.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return env, nil
}

func isGitHubTool(toolName string) bool {
	return strings.HasPrefix(toolName, "github.com")
}
