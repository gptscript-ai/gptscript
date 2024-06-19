package judge

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3gen"
	openai "github.com/gptscript-ai/chat-completion-client"
)

const instructions = `When given JSON objects that conform to the following JSONSchema:

%s

Determine if "actual" is equal to "expected" based on the comparison constraints described by "criteria".
"actual" is considered equal to "expected" if and only if the all of the constraints described by "criteria" are satisfied.

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
      "description": "The reasoning used to come to the determination, that points out all instances where the given criteria was violated"
    }
  },
  "required": [
    "equal",
    "reasoning"
  ]
}

Your responses are concise and include only the json object described above.
`

type Judge[T any] struct {
	client       *openai.Client
	instructions string
}

type comparison[T any] struct {
	Expected T      `json:"expected"`
	Actual   T      `json:"actual"`
	Criteria string `json:"criteria"`
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

	schemaJSON, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONSchema for %T: %w", new(T), err)
	}

	return &Judge[T]{
		client:       client,
		instructions: fmt.Sprintf(instructions, schemaJSON),
	}, nil
}

func (j *Judge[T]) Equal(ctx context.Context, expected, actual T, criteria string) (equal bool, reasoning string, err error) {
	comparisonJSON, err := json.MarshalIndent(&comparison[T]{
		Expected: expected,
		Actual:   actual,
		Criteria: criteria,
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
				Content: j.instructions,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(comparisonJSON),
			},
		},
	}
	response, err := j.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return false, "", fmt.Errorf("failed to make judge chat completion request: %w", err)
	}

	if len(response.Choices) < 1 {
		return false, "", fmt.Errorf("judge chat completion request returned no choices")
	}

	var equality ruling
	if err := json.Unmarshal([]byte(response.Choices[0].Message.Content), &equality); err != nil {
		return false, "", fmt.Errorf("failed to unmarshal judge ruling: %w", err)
	}

	return equality.Equal, equality.Reasoning, nil
}
