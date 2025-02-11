package openai

import (
	"testing"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/hexops/valast"
)

func TestTextToMultiContent(t *testing.T) {
	autogold.Expect([]openai.ChatMessagePart{{
		Type: "text",
		Text: "hi\ndata:image/png;base64,xxxxx\n",
	}}).Equal(t, textToMultiContent("hi\ndata:image/png;base64,xxxxx\n"))

	autogold.Expect([]openai.ChatMessagePart{
		{
			Type: "text",
			Text: "hi",
		},
		{
			Type:     "image_url",
			ImageURL: &openai.ChatMessageImageURL{URL: "data:image/png;base64,xxxxx"},
		},
	}).Equal(t, textToMultiContent("hi\ndata:image/png;base64,xxxxx"))

	autogold.Expect([]openai.ChatMessagePart{{
		Type:     "image_url",
		ImageURL: &openai.ChatMessageImageURL{URL: "data:image/png;base64,xxxxx"},
	}}).Equal(t, textToMultiContent("data:image/png;base64,xxxxx"))

	autogold.Expect([]openai.ChatMessagePart{
		{
			Type: "text",
			Text: "\none\ntwo",
		},
		{
			Type:     "image_url",
			ImageURL: &openai.ChatMessageImageURL{URL: "data:image/png;base64,xxxxx"},
		},
		{
			Type:     "image_url",
			ImageURL: &openai.ChatMessageImageURL{URL: "data:image/png;base64,yyyyy"},
		},
	}).Equal(t, textToMultiContent("\none\ntwo\ndata:image/png;base64,xxxxx\ndata:image/png;base64,yyyyy"))
}

func Test_appendMessage(t *testing.T) {
	autogold.Expect(types.CompletionMessage{Content: []types.ContentPart{
		{ToolCall: &types.CompletionToolCall{
			Index: valast.Ptr(0),
			Function: types.CompletionFunctionCall{
				Name:      "foo",
				Arguments: "bar",
			},
		}},
		{ToolCall: &types.CompletionToolCall{
			Index: valast.Ptr(1),
			Function: types.CompletionFunctionCall{
				Name:      "foo",
				Arguments: "bar",
			},
		}},
	}}).Equal(t, appendMessage(types.CompletionMessage{}, openai.ChatCompletionStreamResponse{
		Choices: []openai.ChatCompletionStreamChoice{
			{
				Delta: openai.ChatCompletionStreamChoiceDelta{
					ToolCalls: []openai.ToolCall{
						{
							Function: openai.FunctionCall{
								Name:      "foo",
								Arguments: "bar",
							},
						},
						{
							Function: openai.FunctionCall{
								Name:      "foo",
								Arguments: "bar",
							},
						},
					},
				},
			},
		},
	}))
}
