package tester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

type Client struct {
	t      *testing.T
	id     int
	result []Result
}

type Result struct {
	Text string
	Func types.CompletionFunctionCall
	Err  error
}

func (c *Client) Call(_ context.Context, messageRequest types.CompletionRequest, _ chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	msgData, err := json.MarshalIndent(messageRequest, "", "  ")
	require.NoError(c.t, err)

	c.id++
	if c.id == 0 {
		autogold.ExpectFile(c.t, string(msgData))
	} else {
		c.t.Run(fmt.Sprintf("call%d", c.id), func(t *testing.T) {
			autogold.ExpectFile(t, string(msgData))
		})
	}
	if len(c.result) == 0 {
		return &types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeAssistant,
			Content: types.Text(fmt.Sprintf("TEST RESULT CALL: %d", c.id)),
		}, nil
	}

	result := c.result[0]
	c.result = c.result[1:]

	if result.Err != nil {
		return nil, result.Err
	}

	for i, tool := range messageRequest.Tools {
		if tool.Function.Name == result.Func.Name {
			return &types.CompletionMessage{
				Role: types.CompletionMessageRoleTypeAssistant,
				Content: []types.ContentPart{
					{
						ToolCall: &types.CompletionToolCall{
							Index: &i,
							ID:    fmt.Sprintf("call_%d", c.id),
							Function: types.CompletionFunctionCall{
								Name:      tool.Function.Name,
								Arguments: result.Func.Arguments,
							},
						},
					},
				},
			}, nil
		}
	}

	if result.Func.Name != "" {
		c.t.Fatalf("failed to find tool %s", result.Func.Name)
	}

	return &types.CompletionMessage{
		Role:    types.CompletionMessageRoleTypeAssistant,
		Content: types.Text(result.Text),
	}, nil
}

type Runner struct {
	*runner.Runner

	Client *Client
}

func (r *Runner) RunDefault() string {
	r.Client.t.Helper()
	result, err := r.Run("", "")
	require.NoError(r.Client.t, err)
	return result
}

func (r *Runner) Run(script, input string) (string, error) {
	if script == "" {
		script = "test.gpt"
	}
	prg, err := loader.Program(context.Background(), filepath.Join(".", "testdata", r.Client.t.Name(), script), "")
	if err != nil {
		return "", err
	}

	return r.Runner.Run(context.Background(), prg, os.Environ(), input)
}

func (r *Runner) RespondWith(result ...Result) {
	r.Client.result = append(r.Client.result, result...)
}

func NewRunner(t *testing.T) *Runner {
	t.Helper()

	c := &Client{
		t: t,
	}

	run, err := runner.New(c)
	require.NoError(t, err)

	return &Runner{
		Runner: run,
		Client: c,
	}
}
