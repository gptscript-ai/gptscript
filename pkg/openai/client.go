package openai

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/acorn-io/gptscript/pkg/cache"
	"github.com/acorn-io/gptscript/pkg/hash"
	"github.com/acorn-io/gptscript/pkg/types"
	"github.com/acorn-io/gptscript/pkg/vision"
	"github.com/sashabaranov/go-openai"
)

const (
	DefaultVisionModel     = openai.GPT4VisionPreview
	DefaultModel           = openai.GPT4TurboPreview
	DefaultMaxTokens       = 1024
	DefaultPromptParameter = "defaultPromptParameter"
)

var (
	key = os.Getenv("OPENAI_API_KEY")
	url = os.Getenv("OPENAI_URL")
)

type Client struct {
	c     *openai.Client
	cache *cache.Client
}

func NewClient(cache *cache.Client) (*Client, error) {
	if url == "" {
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY env var is not set")
		}
		return &Client{
			c:     openai.NewClient(key),
			cache: cache,
		}, nil
	}

	cfg := openai.DefaultConfig(key)
	cfg.BaseURL = url
	return &Client{
		c:     openai.NewClientWithConfig(cfg),
		cache: cache,
	}, nil
}

func (c *Client) cacheKey(request openai.ChatCompletionRequest) string {
	return hash.Encode(request)
}

func (c *Client) fromCache(ctx context.Context, messageRequest types.CompletionRequest, request openai.ChatCompletionRequest) (result []openai.ChatCompletionStreamResponse, _ bool, _ error) {
	if messageRequest.Cache != nil && !*messageRequest.Cache {
		return nil, false, nil
	}

	cache, found, err := c.cache.Get(ctx, c.cacheKey(request))
	if err != nil {
		return nil, false, err
	} else if !found {
		return nil, false, nil
	}

	gz, err := gzip.NewReader(bytes.NewReader(cache))
	if err != nil {
		return nil, false, err
	}
	return result, true, json.NewDecoder(gz).Decode(&result)
}

func toToolCall(call types.CompletionToolCall) openai.ToolCall {
	return openai.ToolCall{
		Index: call.Index,
		ID:    call.ID,
		Type:  openai.ToolType(call.Type),
		Function: openai.FunctionCall{
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		},
	}
}

func toMessages(ctx context.Context, cache *cache.Client, request types.CompletionRequest) (result []openai.ChatCompletionMessage, err error) {
	for _, message := range request.Messages {
		if request.Vision {
			message, err = vision.ToVisionMessage(ctx, cache, message)
			if err != nil {
				return nil, err
			}
		}

		chatMessage := openai.ChatCompletionMessage{
			Role: string(message.Role),
		}

		if message.ToolCall != nil {
			chatMessage.ToolCallID = message.ToolCall.ID
		}

		for _, content := range message.Content {
			if content.ToolCall != nil {
				chatMessage.ToolCalls = append(chatMessage.ToolCalls, toToolCall(*content.ToolCall))
			}
			if content.Image != nil {
				url, err := vision.ImageToURL(ctx, cache, request.Vision, *content.Image)
				if err != nil {
					return nil, err
				}
				if request.Vision {
					chatMessage.MultiContent = append(chatMessage.MultiContent, openai.ChatMessagePart{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: url,
						},
					})
				} else {
					chatMessage.MultiContent = append(chatMessage.MultiContent, openai.ChatMessagePart{
						Type: openai.ChatMessagePartTypeText,
						Text: fmt.Sprintf("Image URL %s", url),
					})
				}
			}
			if content.Text != "" {
				chatMessage.MultiContent = append(chatMessage.MultiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: content.Text,
				})
			}
		}

		if len(chatMessage.MultiContent) == 1 && chatMessage.MultiContent[0].Type == openai.ChatMessagePartTypeText {
			if chatMessage.MultiContent[0].Text == "." || chatMessage.MultiContent[0].Text == "{}" {
				continue
			}
			chatMessage.Content = chatMessage.MultiContent[0].Text
			chatMessage.MultiContent = nil

			if strings.Contains(chatMessage.Content, DefaultPromptParameter) && strings.HasPrefix(chatMessage.Content, "{") {
				data := map[string]any{}
				if err := json.Unmarshal([]byte(chatMessage.Content), &data); err == nil && len(data) == 1 {
					if v, _ := data[DefaultPromptParameter].(string); v != "" {
						chatMessage.Content = v
					}
				}
			}
		}

		result = append(result, chatMessage)
	}
	return
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionMessage) (*types.CompletionMessage, error) {
	msgs, err := toMessages(ctx, c.cache, messageRequest)
	if err != nil {
		return nil, err
	}

	if len(msgs) == 0 {
		return nil, fmt.Errorf("invalid request, no messages to send to OpenAI")
	}

	request := openai.ChatCompletionRequest{
		Model:     messageRequest.Model,
		Messages:  msgs,
		MaxTokens: messageRequest.MaxToken,
	}

	if messageRequest.JSONResponse {
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}

	if request.Model == "" {
		if messageRequest.Vision {
			request.Model = DefaultVisionModel
		} else {
			request.Model = DefaultModel
		}
	}

	if request.MaxTokens == 0 {
		request.MaxTokens = DefaultMaxTokens
	}

	if !messageRequest.Vision {
		for _, tool := range messageRequest.Tools {
			params := tool.Function.Parameters
			if params != nil && params.Type == "object" && params.Properties == nil {
				params.Properties = map[string]types.Property{}
			}
			request.Tools = append(request.Tools, openai.Tool{
				Type: openai.ToolType(tool.Type),
				Function: openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  params,
				},
			})
		}
	}

	request.Seed = ptr(hash.Seed(request))
	response, ok, err := c.fromCache(ctx, messageRequest, request)
	if err != nil {
		return nil, err
	} else if !ok {
		response, err = c.call(ctx, request, status)
		if err != nil {
			return nil, err
		}
	}

	result := types.CompletionMessage{}
	for _, response := range response {
		result = appendMessage(result, response)
	}

	return &result, nil
}

func appendMessage(msg types.CompletionMessage, response openai.ChatCompletionStreamResponse) types.CompletionMessage {
	if len(response.Choices) == 0 {
		return msg
	}

	delta := response.Choices[0].Delta
	msg.Role = types.CompletionMessageRoleType(override(string(msg.Role), delta.Role))

	for _, tool := range delta.ToolCalls {
		if tool.Index == nil {
			continue
		}
		idx := *tool.Index
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
		tc.ToolCall.Type = types.CompletionToolType(override(string(tc.ToolCall.Type), string(tool.Type)))
		tc.ToolCall.Function.Name += tool.Function.Name
		tc.ToolCall.Function.Arguments += tool.Function.Arguments

		msg.Content[idx] = tc
	}

	if delta.Content != "" {
		found := false
		for i, content := range msg.Content {
			if content.ToolCall != nil || content.Image != nil {
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

func (c *Client) store(ctx context.Context, key string, responses []openai.ChatCompletionStreamResponse) error {
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	err := json.NewEncoder(gz).Encode(responses)
	if err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return c.cache.Store(ctx, key, buf.Bytes())
}

func (c *Client) call(ctx context.Context, request openai.ChatCompletionRequest, partial chan<- types.CompletionMessage) (responses []openai.ChatCompletionStreamResponse, _ error) {
	cacheKey := c.cacheKey(request)
	request.Stream = true

	slog.Debug("calling openai", "message", request.Messages)
	stream, err := c.c.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			return responses, c.store(ctx, cacheKey, responses)
		} else if err != nil {
			return nil, err
		}
		if len(response.Choices) > 0 {
			slog.Debug("stream", "content", response.Choices[0].Delta.Content)
			if partial != nil {
				partial <- types.CompletionMessage{
					Role:    types.CompletionMessageRoleType(response.Choices[0].Delta.Role),
					Content: types.Text(response.Choices[0].Delta.Content),
				}
			}
		}
		responses = append(responses, response)
	}
}

func ptr[T any](v T) *T {
	return &v
}
