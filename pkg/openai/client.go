package openai

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/cache"
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
	TooLongMessage  = "Error: tool call output is too long"
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

func (c *Client) ProxyInfo([]string) (token, urlBase string) {
	if c.invalidAuth {
		return "", ""
	}
	return c.c.GetAPIKeyAndBaseURL()
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

	for _, model := range models {
		if model.ID == modelName {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) ListModels(ctx context.Context, providers ...string) ([]openai.Model, error) {
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
	sort.Slice(models.Models, func(i, j int) bool {
		return models.Models[i].ID < models.Models[j].ID
	})
	return models.Models, nil
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

func (c *Client) fromCache(ctx context.Context, messageRequest types.CompletionRequest, request openai.ChatCompletionRequest) (result types.CompletionMessage, _ bool, _ error) {
	if !messageRequest.GetCache() {
		return types.CompletionMessage{}, false, nil
	}
	found, err := c.cache.Get(ctx, c.cacheKey(request), &result)
	if err != nil {
		return types.CompletionMessage{}, false, err
	} else if !found {
		return types.CompletionMessage{}, false, nil
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
				chatMessage.MultiContent = append(chatMessage.MultiContent, textToMultiContent(content.Text)...)
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

const imagePrefix = "data:image/png;base64,"

func textToMultiContent(text string) []openai.ChatMessagePart {
	var chatParts []openai.ChatMessagePart
	parts := strings.Split(text, "\n")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasPrefix(parts[i], imagePrefix) {
			chatParts = append(chatParts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL: parts[i],
				},
			})
			parts = parts[:i]
		} else {
			break
		}
	}
	if len(parts) > 0 {
		chatParts = append(chatParts, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: strings.Join(parts, "\n"),
		})
	}

	slices.Reverse(chatParts)
	return chatParts
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, env []string, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	if err := c.ValidAuth(); err != nil {
		if err := c.RetrieveAPIKey(ctx, env); err != nil {
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
		// Check the last message. If it is from a tool call, and if it takes up more than 80% of the budget on its own, reject it.
		lastMessage := msgs[len(msgs)-1]
		if lastMessage.Role == string(types.CompletionMessageRoleTypeTool) && countMessage(lastMessage) > int(float64(getBudget(messageRequest.MaxTokens))*0.8) {
			// We need to update it in the msgs slice for right now and in the messageRequest for future calls.
			msgs[len(msgs)-1].Content = TooLongMessage
			messageRequest.Messages[len(messageRequest.Messages)-1].Content = types.Text(TooLongMessage)
		}

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

	toolMapping := map[string]string{}
	for _, tool := range messageRequest.Tools {
		var params any = tool.Function.Parameters
		if tool.Function.Parameters == nil || len(tool.Function.Parameters.Properties) == 0 {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}

		if tool.Function.ToolID != "" {
			toolMapping[tool.Function.Name] = tool.Function.ToolID
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
		Request: map[string]any{
			"chatCompletion": request,
			"toolMapping":    toolMapping,
		},
	}

	var cacheResponse bool
	if c.setSeed {
		request.Seed = ptr(c.seed(request))
		request.StreamOptions = &openai.StreamOptions{
			IncludeUsage: true,
		}
	}
	result, ok, err := c.fromCache(ctx, messageRequest, request)
	if err != nil {
		return nil, err
	} else if !ok {
		result, err = c.call(ctx, request, id, env, status)

		// If we got back a context length exceeded error, keep retrying and shrinking the message history until we pass.
		var apiError *openai.APIError
		if errors.As(err, &apiError) && apiError.Code == "context_length_exceeded" && messageRequest.Chat {
			// Decrease maxTokens by 10% to make garbage collection more aggressive.
			// The retry loop will further decrease maxTokens if needed.
			maxTokens := decreaseTenPercent(messageRequest.MaxTokens)
			result, err = c.contextLimitRetryLoop(ctx, request, id, env, maxTokens, status)
		}
		if err != nil {
			return nil, err
		}
	} else {
		cacheResponse = true
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
		Response:     result,
		Usage:        result.Usage,
		Cached:       cacheResponse,
	}

	return &result, nil
}

func (c *Client) contextLimitRetryLoop(ctx context.Context, request openai.ChatCompletionRequest, id string, env []string, maxTokens int, status chan<- types.CompletionStatus) (types.CompletionMessage, error) {
	var (
		response types.CompletionMessage
		err      error
	)

	for range 10 { // maximum 10 tries
		// Try to drop older messages again, with a decreased max tokens.
		request.Messages = dropMessagesOverCount(maxTokens, request.Messages)
		response, err = c.call(ctx, request, id, env, status)
		if err == nil {
			return response, nil
		}

		var apiError *openai.APIError
		if errors.As(err, &apiError) && apiError.Code == "context_length_exceeded" {
			// Decrease maxTokens and try again
			maxTokens = decreaseTenPercent(maxTokens)
			continue
		}
		return types.CompletionMessage{}, err
	}

	return types.CompletionMessage{}, err
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
		// OpenAI like to sometimes add these prefix because it's confused
		tc.ToolCall.Function.Name = strings.TrimPrefix(tc.ToolCall.Function.Name, "namespace.")
		tc.ToolCall.Function.Name = strings.TrimPrefix(tc.ToolCall.Function.Name, "@")
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

func (c *Client) call(ctx context.Context, request openai.ChatCompletionRequest, transactionID string, env []string, partial chan<- types.CompletionStatus) (types.CompletionMessage, error) {
	streamResponse := os.Getenv("GPTSCRIPT_INTERNAL_OPENAI_STREAMING") != "false"

	partial <- types.CompletionStatus{
		CompletionID: transactionID,
		PartialResponse: &types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeAssistant,
			Content: types.Text("Waiting for model response..."),
		},
	}

	var (
		headers          map[string]string
		modelProviderEnv []string
		retryOpts        = []openai.RetryOptions{
			{
				Retries:        5,
				RetryAboveCode: 499,        // 5xx errors
				RetryCodes:     []int{429}, // 429 Too Many Requests (ratelimit)
			},
		}
	)
	for _, e := range env {
		if strings.HasPrefix(e, "GPTSCRIPT_MODEL_PROVIDER_") {
			modelProviderEnv = append(modelProviderEnv, e)
		} else if strings.HasPrefix(e, "GPTSCRIPT_DISABLE_RETRIES") {
			retryOpts = nil
		}
	}

	if len(modelProviderEnv) > 0 {
		headers = map[string]string{
			"X-GPTScript-Env": strings.Join(modelProviderEnv, ","),
		}
	}

	slog.Debug("calling openai", "message", request.Messages)

	if !streamResponse {
		request.StreamOptions = nil
		resp, err := c.c.CreateChatCompletion(ctx, request, headers, retryOpts...)
		if err != nil {
			return types.CompletionMessage{}, err
		}
		return appendMessage(types.CompletionMessage{}, openai.ChatCompletionStreamResponse{
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
		}), nil
	}

	stream, err := c.c.CreateChatCompletionStream(ctx, request, headers, retryOpts...)
	if err != nil {
		return types.CompletionMessage{}, err
	}
	defer stream.Close()

	var (
		partialMessage types.CompletionMessage
		start          = time.Now()
		last           []string
	)
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			return partialMessage, c.cache.Store(ctx, c.cacheKey(request), partialMessage)
		} else if err != nil {
			return types.CompletionMessage{}, err
		}
		partialMessage = appendMessage(partialMessage, response)
		if partial != nil {
			if time.Since(start) > 100*time.Millisecond {
				last = last[:0]
				partial <- types.CompletionStatus{
					CompletionID:    transactionID,
					PartialResponse: &partialMessage,
				}
				start = time.Now()
			}
		}
	}
}

func (c *Client) RetrieveAPIKey(ctx context.Context, env []string) error {
	k, err := prompt.GetModelProviderCredential(ctx, c.credStore, BuiltinCredName, "OPENAI_API_KEY", "Please provide your OpenAI API key:", env)
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
