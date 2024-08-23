package judge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3gen"
	openai "github.com/gptscript-ai/chat-completion-client"
)

const instructions = `"actual" is considered equivalent to "expected" if and only if the following rules are satisfied:

%s

When given JSON objects that conform to the following JSONSchema:

%s

Determine if "actual" is considered equivalent to "expected".

After making a determination, respond with a JSON object that conforms to the following JSONSchema:

{
  "name": "ruling",
  "type": "object",
  "properties": {
    "equal": {
      "type": "boolean",
        "description": "Set to true if and only if actual is considered equal to expected."
      },
    "reasoning": {
      "type": "string",
      "description": "The reasoning used to come to the determination"
    }
  },
  "required": [
    "equal",
    "reasoning"
  ]
}

If you determine actual and expected are not equivalent, include a diff of the parts of actual and expected that are not equivalent in the reasoning field of your response.

Your responses are concise and include only the json object described above.
`

type Judge[T any] struct {
	client           *openai.Client
	comparisonSchema string
}

type comparison[T any] struct {
	Expected T `json:"expected"`
	Actual   T `json:"actual"`
}

type ruling struct {
	Equal     bool   `json:"equal"`
	Reasoning string `json:"reasoning"`
}

func New[T any](client *openai.Client) (*Judge[T], error) {
	schema, err := openapi3gen.NewSchemaRefForValue(
		new(comparison[T]),
		nil,
		openapi3gen.CreateComponentSchemas(
			openapi3gen.ExportComponentSchemasOptions{
				ExportComponentSchemas: true,
				ExportGenerics:         false,
			}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSONSchema for %T: %w", new(T), err)
	}

	marshaled, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONSchema for %T: %w", new(T), err)
	}

	return &Judge[T]{
		client:           client,
		comparisonSchema: string(marshaled),
	}, nil
}

func (j *Judge[T]) Equal(ctx context.Context, expected, actual T, criteria string) (equal bool, reasoning string, err error) {
	comparisonJSON, err := json.MarshalIndent(&comparison[T]{
		Expected: expected,
		Actual:   actual,
	}, "", "    ")
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal judge testcase JSON: %w", err)
	}

	request := openai.ChatCompletionRequest{
		Model:       openai.GPT4o,
		Temperature: new(float32),
		N:           1,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf(instructions, criteria, j.comparisonSchema),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(comparisonJSON),
			},
		},
	}
	response, err := j.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return false, "", fmt.Errorf("failed to create chat completion request: %w", err)
	}

	if len(response.Choices) < 1 {
		return false, "", fmt.Errorf("chat completion request returned no choices")
	}

	var equality ruling
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &equality); err != nil {
		return false, "", fmt.Errorf("failed to unmarshal judge ruling: %w", err)
	}

	return equality.Equal, equality.Reasoning, nil
}
