package openai

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	gcontext "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/system"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

const (
	DefaultModel    = openai.GPT4o
	BuiltinCredName = "sys.openai"
)

var (
	key = os.Getenv("OPENAI_API_KEY")
	url = os.Getenv("OPENAI_BASE_URL")
	log = mvl.Package()
)

type InvalidAuthError struct{}

func (InvalidAuthError) Error() string {
	return "OPENAI_API_KEY is not set. Please set the OPENAI_API_KEY environment variable"
}

type Client struct {
	defaultModel string
	c            *openai.Client
	cache        *cache.Client
	invalidAuth  bool
	cacheKeyBase string
	setSeed      bool
	credStore    credentials.CredentialStore
}

type Options struct {
	BaseURL      string `usage:"OpenAI base URL" name:"openai-base-url" env:"OPENAI_BASE_URL"`
	APIKey       string `usage:"OpenAI API KEY" name:"openai-api-key" env:"OPENAI_API_KEY"`
	OrgID        string `usage:"OpenAI organization ID" name:"openai-org-id" env:"OPENAI_ORG_ID"`
	DefaultModel string `usage:"Default LLM model to use" default:"gpt-4o"`
	ConfigFile   string `usage:"Path to GPTScript config file" name:"config"`
	SetSeed      bool   `usage:"-"`
	CacheKey     string `usage:"-"`
	Cache        *cache.Client
}

func Complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.BaseURL = types.FirstSet(opt.BaseURL, result.BaseURL)
		result.APIKey = types.FirstSet(opt.APIKey, result.APIKey)
		result.OrgID = types.FirstSet(opt.OrgID, result.OrgID)
		result.Cache = types.FirstSet(opt.Cache, result.Cache)
		result.DefaultModel = types.FirstSet(opt.DefaultModel, result.DefaultModel)
		result.SetSeed = types.FirstSet(opt.SetSeed, result.SetSeed)
		result.CacheKey = types.FirstSet(opt.CacheKey, result.CacheKey)
	}

	return result
}

func complete(opts ...Options) (Options, error) {
	var err error
	result := Complete(opts...)
	if result.Cache == nil {
		result.Cache, err = cache.New(cache.Options{
			DisableCache: true,
		})
	}

	if result.BaseURL == "" && url != "" {
		result.BaseURL = url
	}

	if result.APIKey == "" && key != "" {
		result.APIKey = key
	}

	return result, err
}

func NewClient(ctx context.Context, credStore credentials.CredentialStore, opts ...Options) (*Client, error) {
	opt, err := complete(opts...)
	if err != nil {
		return nil, err
	}

	// If the API key is not set, try to get it from the cred store
	if opt.APIKey == "" && opt.BaseURL == "" {
		cred, exists, err := credStore.Get(ctx, BuiltinCredName)
		if err != nil {
			return nil, err
		}
		if exists {
			opt.APIKey = cred.Env["OPENAI_API_KEY"]
		}
	}

	cfg := openai.DefaultConfig(opt.APIKey)
	cfg.BaseURL = types.FirstSet(opt.BaseURL, cfg.BaseURL)
	cfg.OrgID = types.FirstSet(opt.OrgID, cfg.OrgID)

	cacheKeyBase := opt.CacheKey
	if cacheKeyBase == "" {
		cacheKeyBase = hash.ID(opt.APIKey, opt.BaseURL)
	}

	return &Client{
		c:            openai.NewClientWithConfig(cfg),
		cache:        opt.Cache,
		defaultModel: opt.DefaultModel,
		cacheKeyBase: cacheKeyBase,
		invalidAuth:  opt.APIKey == "" && opt.BaseURL == "",
		setSeed:      opt.SetSeed,
		credStore:    credStore,
	}, nil
}

func (c *Client) ValidAuth() error {
	if c.invalidAuth {
		return InvalidAuthError{}
	}
	return nil
}

func (c *Client) Supports(ctx context.Context, modelName string) (bool, error) {
	models, err := c.ListModels(ctx)
	if err != nil {
		return false, err
	}

	if len(models) == 0 {
		// We got no models back, which means our auth is invalid.
		return false, InvalidAuthError{}
	}

	return slices.Contains(models, modelName), nil
}

func (c *Client) ListModels(ctx context.Context, providers ...string) (result []string, _ error) {
	// Only serve if providers is empty or "" is in the list
	if len(providers) != 0 && !slices.Contains(providers, "") {
		return nil, nil
	}

	// If auth is invalid, we just want to return nothing.
	// Returning an InvalidAuthError here will lead to cases where the user is prompted to enter their OpenAI key,
	// even when we don't want them to be prompted.
	// So the UX we settled on is that no models get printed if the user does gptscript --list-models
	// without having provided their key through the environment variable or the creds store.
	if err := c.ValidAuth(); err != nil {
		return nil, nil
	}

	models, err := c.c.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	for _, model := range models.Models {
		result = append(result, model.ID)
	}
	sort.Strings(result)
	return result, nil
}

func (c *Client) cacheKey(request openai.ChatCompletionRequest) any {
	return map[string]any{
		"base":    c.cacheKeyBase,
		"request": request,
	}
}

func (c *Client) seed(request openai.ChatCompletionRequest) int {
	newRequest := request
	newRequest.Messages = nil

	for _, msg := range request.Messages {
		newMsg := msg
		newMsg.ToolCalls = nil
		newMsg.ToolCallID = ""

		for _, tool := range msg.ToolCalls {
			tool.ID = ""
			newMsg.ToolCalls = append(newMsg.ToolCalls, tool)
		}

		newRequest.Messages = append(newRequest.Messages, newMsg)
	}
	return hash.Seed(newRequest)
}

func (c *Client) fromCache(ctx context.Context, messageRequest types.CompletionRequest, request openai.ChatCompletionRequest) (result []openai.ChatCompletionStreamResponse, _ bool, _ error) {
	if !messageRequest.GetCache() {
		return nil, false, nil
	}
	found, err := c.cache.Get(ctx, c.cacheKey(request), &result)
	if err != nil {
		return nil, false, err
	} else if !found {
		return nil, false, nil
	}
	return result, true, nil
}

func toToolCall(call types.CompletionToolCall) openai.ToolCall {
	return openai.ToolCall{
		ID:   call.ID,
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		},
	}
}

func toMessages(request types.CompletionRequest, compat bool) (result []openai.ChatCompletionMessage, err error) {
	var (
		systemPrompts []string
		msgs          []types.CompletionMessage
	)

	if !compat && (request.InternalSystemPrompt == nil || *request.InternalSystemPrompt) {
		systemPrompts = append(systemPrompts, system.InternalSystemPrompt)
	}

	for _, message := range request.Messages {
		if message.Role == types.CompletionMessageRoleTypeSystem {
			systemPrompts = append(systemPrompts, message.Content[0].Text)
			continue
		}
		msgs = append(msgs, message)
	}

	if len(systemPrompts) > 0 {
		msgs = slices.Insert(msgs, 0, types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeSystem,
			Content: types.Text(strings.Join(systemPrompts, "\n")),
		})
	}

	for _, message := range msgs {
		chatMessage := openai.ChatCompletionMessage{
			Role: string(message.Role),
		}

		if message.ToolCall != nil {
			chatMessage.ToolCallID = message.ToolCall.ID
			// This field is not documented but specifically Azure thinks it should be set
			chatMessage.Name = message.ToolCall.Function.Name
		}

		for _, content := range message.Content {
			if content.ToolCall != nil {
				chatMessage.ToolCalls = append(chatMessage.ToolCalls, toToolCall(*content.ToolCall))
			}
			if content.Text != "" {
				chatMessage.MultiContent = append(chatMessage.MultiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: content.Text,
				})
			}
		}

		if len(chatMessage.MultiContent) == 1 && chatMessage.MultiContent[0].Type == openai.ChatMessagePartTypeText {
			if !request.Chat && strings.TrimSpace(chatMessage.MultiContent[0].Text) == "{}" {
				continue
			}
			chatMessage.Content = chatMessage.MultiContent[0].Text
			chatMessage.MultiContent = nil

			if prompt, ok := system.IsDefaultPrompt(chatMessage.Content); ok {
				chatMessage.Content = prompt
			}
		}

		result = append(result, chatMessage)
	}

	return
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	if err := c.ValidAuth(); err != nil {
		if err := c.RetrieveAPIKey(ctx); err != nil {
			return nil, err
		}
	}

	if messageRequest.Model == "" {
		messageRequest.Model = c.defaultModel
	}

	msgs, err := toMessages(messageRequest, !c.setSeed)
	if err != nil {
		return nil, err
	}

	if messageRequest.Chat {
		msgs = dropMessagesOverCount(messageRequest.MaxTokens, msgs)
	}

	if len(msgs) == 0 {
		log.Errorf("invalid request, no messages to send to LLM")
		return &types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeAssistant,
			Content: types.Text(""),
		}, nil
	}

	request := openai.ChatCompletionRequest{
		Model:     messageRequest.Model,
		Messages:  msgs,
		MaxTokens: messageRequest.MaxTokens,
	}

	if messageRequest.Temperature == nil {
		request.Temperature = new(float32)
	} else {
		request.Temperature = messageRequest.Temperature
	}

	if messageRequest.JSONResponse {
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	for _, tool := range messageRequest.Tools {
		var params any = tool.Function.Parameters
		if tool.Function.Parameters == nil || len(tool.Function.Parameters.Properties) == 0 {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}

		request.Tools = append(request.Tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  params,
			},
		})
	}

	id := counter.Next()
	status <- types.CompletionStatus{
		CompletionID: id,
		Request:      request,
	}

	var cacheResponse bool
	if c.setSeed {
		request.Seed = ptr(c.seed(request))
		request.StreamOptions = &openai.StreamOptions{
			IncludeUsage: true,
		}
	}
	response, ok, err := c.fromCache(ctx, messageRequest, request)
	if err != nil {
		return nil, err
	} else if !ok {
		response, err = c.call(ctx, request, id, status)
		if err != nil {
			return nil, err
		}
	} else {
		cacheResponse = true
	}

	result := types.CompletionMessage{}
	for _, response := range response {
		result = appendMessage(result, response)
	}

	for i, content := range result.Content {
		if content.ToolCall != nil && content.ToolCall.ID == "" {
			content.ToolCall.ID = "call_" + hash.ID(content.ToolCall.Function.Name, content.ToolCall.Function.Arguments)[:8]
			result.Content[i] = content
		}
	}

	if result.Role == "" {
		result.Role = types.CompletionMessageRoleTypeAssistant
	}

	if cacheResponse {
		result.Usage = types.Usage{}
	}

	status <- types.CompletionStatus{
		CompletionID: id,
		Chunks:       response,
		Response:     result,
		Usage:        result.Usage,
		Cached:       cacheResponse,
	}

	return &result, nil
}

func appendMessage(msg types.CompletionMessage, response openai.ChatCompletionStreamResponse) types.CompletionMessage {
	msg.Usage.CompletionTokens = types.FirstSet(msg.Usage.CompletionTokens, response.Usage.CompletionTokens)
	msg.Usage.PromptTokens = types.FirstSet(msg.Usage.PromptTokens, response.Usage.PromptTokens)
	msg.Usage.TotalTokens = types.FirstSet(msg.Usage.TotalTokens, response.Usage.TotalTokens)

	if len(response.Choices) == 0 {
		return msg
	}

	delta := response.Choices[0].Delta
	msg.Role = types.CompletionMessageRoleType(override(string(msg.Role), delta.Role))

	for i, tool := range delta.ToolCalls {
		idx := i
		if tool.Index != nil {
			idx = *tool.Index
		}
		for len(msg.Content)-1 < idx {
			msg.Content = append(msg.Content, types.ContentPart{
				ToolCall: &types.CompletionToolCall{
					Index: ptr(len(msg.Content)),
				},
			})
		}

		tc := msg.Content[idx]
		if tc.ToolCall == nil {
			tc.ToolCall = &types.CompletionToolCall{}
		}
		if tool.Index != nil {
			tc.ToolCall.Index = tool.Index
		}
		tc.ToolCall.ID = override(tc.ToolCall.ID, tool.ID)
		if tc.ToolCall.Function.Name != tool.Function.Name {
			tc.ToolCall.Function.Name += tool.Function.Name
		}
		// OpenAI like to sometimes add this prefix for no good reason
		tc.ToolCall.Function.Name = strings.TrimPrefix(tc.ToolCall.Function.Name, "namespace.")
		tc.ToolCall.Function.Arguments += tool.Function.Arguments

		msg.Content[idx] = tc
	}

	if delta.Content != "" {
		found := false
		for i, content := range msg.Content {
			if content.ToolCall != nil {
				continue
			}
			msg.Content[i] = types.ContentPart{
				Text: msg.Content[i].Text + delta.Content,
			}
			found = true
			break
		}
		if !found {
			msg.Content = append(msg.Content, types.ContentPart{
				Text: delta.Content,
			})
		}
	}

	return msg
}

func override(left, right string) string {
	if right != "" {
		return right
	}
	return left
}

func (c *Client) call(ctx context.Context, request openai.ChatCompletionRequest, transactionID string, partial chan<- types.CompletionStatus) (responses []openai.ChatCompletionStreamResponse, _ error) {
	streamResponse := os.Getenv("GPTSCRIPT_INTERNAL_OPENAI_STREAMING") != "false"

	partial <- types.CompletionStatus{
		CompletionID: transactionID,
		PartialResponse: &types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeAssistant,
			Content: types.Text("Waiting for model response..."),
		},
	}

	slog.Debug("calling openai", "message", request.Messages)

	if !streamResponse {
		request.StreamOptions = nil
		resp, err := c.c.CreateChatCompletion(ctx, request)
		if err != nil {
			return nil, err
		}
		return []openai.ChatCompletionStreamResponse{
			{
				ID:      resp.ID,
				Object:  resp.Object,
				Created: resp.Created,
				Model:   resp.Model,
				Usage:   resp.Usage,
				Choices: []openai.ChatCompletionStreamChoice{
					{
						Index: resp.Choices[0].Index,
						Delta: openai.ChatCompletionStreamChoiceDelta{
							Content:      resp.Choices[0].Message.Content,
							Role:         resp.Choices[0].Message.Role,
							FunctionCall: resp.Choices[0].Message.FunctionCall,
							ToolCalls:    resp.Choices[0].Message.ToolCalls,
						},
						FinishReason: resp.Choices[0].FinishReason,
					},
				},
			},
		}, nil
	}

	stream, err := c.c.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var partialMessage types.CompletionMessage
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			return responses, c.cache.Store(ctx, c.cacheKey(request), responses)
		} else if err != nil {
			return nil, err
		}
		if len(response.Choices) > 0 {
			slog.Debug("stream", "content", response.Choices[0].Delta.Content)
		}
		if partial != nil {
			partialMessage = appendMessage(partialMessage, response)
			partial <- types.CompletionStatus{
				CompletionID:    transactionID,
				PartialResponse: &partialMessage,
			}
		}
		responses = append(responses, response)
	}
}

func (c *Client) RetrieveAPIKey(ctx context.Context) error {
	k, err := prompt.GetModelProviderCredential(ctx, c.credStore, BuiltinCredName, "OPENAI_API_KEY", "Please provide your OpenAI API key:", gcontext.GetEnv(ctx))
	if err != nil {
		return err
	}

	c.c.SetAPIKey(k)
	c.invalidAuth = false
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
