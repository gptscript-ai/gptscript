package tester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/credentials"
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
	Text    string
	Func    types.CompletionFunctionCall
	Content []types.ContentPart
	Err     error
}

func (c *Client) Call(_ context.Context, messageRequest types.CompletionRequest, _ chan<- types.CompletionStatus) (resp *types.CompletionMessage, respErr error) {
	msgData, err := json.MarshalIndent(messageRequest, "", "  ")
	require.NoError(c.t, err)

	c.id++
	if c.id == 0 {
		autogold.ExpectFile(c.t, string(msgData))
	} else {
		c.t.Run(fmt.Sprintf("call%d", c.id), func(t *testing.T) {
			autogold.ExpectFile(t, string(msgData))
		})
		defer func() {
			if respErr == nil {
				c.t.Run(fmt.Sprintf("call%d-resp", c.id), func(t *testing.T) {
					msgData, err := json.MarshalIndent(resp, "", "  ")
					require.NoError(c.t, err)
					autogold.ExpectFile(t, string(msgData))
				})
			}
		}()
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

	if len(result.Content) > 0 {
		msg := &types.CompletionMessage{
			Role: types.CompletionMessageRoleTypeAssistant,
		}
		for _, contentPart := range result.Content {
			if contentPart.ToolCall == nil {
				msg.Content = append(msg.Content, contentPart)
			} else {
				for i, tool := range messageRequest.Tools {
					if contentPart.ToolCall.Function.Name == tool.Function.Name {
						contentPart.ToolCall.Index = &i
						msg.Content = append(msg.Content, contentPart)
					}
				}
			}
		}
		return msg, nil
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

func (r *Runner) Load(script string) (types.Program, error) {
	if script == "" {
		script = "test.gpt"
	}
	return loader.Program(context.Background(), filepath.Join(".", "testdata", r.Client.t.Name(), script), "")
}

func (r *Runner) Run(script, input string) (string, error) {
	prg, err := r.Load(script)
	if err != nil {
		return "", err
	}

	return r.Runner.Run(context.Background(), prg, os.Environ(), input)
}

func (r *Runner) AssertResponded(t *testing.T) {
	t.Helper()
	require.Len(t, r.Client.result, 0)
}

func (r *Runner) RespondWith(result ...Result) {
	r.Client.result = append(r.Client.result, result...)
}

func NewRunner(t *testing.T) *Runner {
	t.Helper()

	c := &Client{
		t: t,
	}

	run, err := runner.New(c, credentials.NoopStore{}, runner.Options{
		Sequential: true,
	})
	require.NoError(t, err)

	return &Runner{
		Runner: run,
		Client: c,
	}
}
