package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func toJSONString(t *testing.T, v interface{}) string {
	t.Helper()
	x, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	return string(x)
}

func TestDualSubChat(t *testing.T) {
	r := tester.NewRunner(t)
	r.RespondWith(tester.Result{
		Content: []types.ContentPart{
			{
				ToolCall: &types.CompletionToolCall{
					ID: "call_1",
					Function: types.CompletionFunctionCall{
						Name:      "chatbot",
						Arguments: "Input to chatbot1",
					},
				},
			},
			{
				ToolCall: &types.CompletionToolCall{
					ID: "call_2",
					Function: types.CompletionFunctionCall{
						Name:      "chatbot2",
						Arguments: "Input to chatbot2",
					},
				},
			},
		},
	}, tester.Result{
		Text: "Assistant Response 1 - from chatbot1",
	}, tester.Result{
		Text: "Assistent Response 2 - from chatbot2",
	})

	prg, err := r.Load("")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, os.Environ(), "User 1")
	require.NoError(t, err)
	r.AssertResponded(t)
	assert.False(t, resp.Done)
	autogold.Expect("Assistant Response 1 - from chatbot1").Equal(t, resp.Content)
	autogold.ExpectFile(t, toJSONString(t, resp), autogold.Name(t.Name()+"/step1"))

	r.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name:      types.ToolNormalizer("sys.chat.finish"),
			Arguments: `{"message":"Chat done"}`,
		},
	})

	resp, err = r.Chat(context.Background(), resp.State, prg, os.Environ(), "User 2")
	require.NoError(t, err)
	r.AssertResponded(t)
	assert.False(t, resp.Done)
	autogold.Expect("Assistent Response 2 - from chatbot2").Equal(t, resp.Content)
	autogold.ExpectFile(t, toJSONString(t, resp), autogold.Name(t.Name()+"/step2"))

	r.RespondWith(tester.Result{
		Text: "Assistant 3",
	})

	resp, err = r.Chat(context.Background(), resp.State, prg, os.Environ(), "User 3")
	require.NoError(t, err)
	r.AssertResponded(t)
	assert.False(t, resp.Done)
	autogold.Expect("Assistant 3").Equal(t, resp.Content)
	autogold.ExpectFile(t, toJSONString(t, resp), autogold.Name(t.Name()+"/step3"))

	r.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name:      types.ToolNormalizer("sys.chat.finish"),
			Arguments: `{"message":"Chat done2"}`,
		},
	}, tester.Result{
		Text: "And we're done",
	})

	resp, err = r.Chat(context.Background(), resp.State, prg, os.Environ(), "User 4")
	require.NoError(t, err)
	r.AssertResponded(t)
	assert.True(t, resp.Done)
	autogold.Expect("And we're done").Equal(t, resp.Content)
	autogold.ExpectFile(t, toJSONString(t, resp), autogold.Name(t.Name()+"/step4"))
}

func TestSubChat(t *testing.T) {
	r := tester.NewRunner(t)
	r.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "chatbot",
		},
	}, tester.Result{
		Text: "Assistant 1",
	}, tester.Result{
		Text: "Assistant 2",
	})

	prg, err := r.Load("")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, os.Environ(), "Hello")
	require.NoError(t, err)

	autogold.Expect(`{
  "done": false,
  "content": "Assistant 1",
  "toolID": "testdata/TestSubChat/test.gpt:6",
  "state": {
    "continuation": {
      "state": {
        "input": "Hello",
        "completion": {
          "Model": "gpt-4-turbo-preview",
          "InternalSystemPrompt": null,
          "Tools": [
            {
              "function": {
                "toolID": "testdata/TestSubChat/test.gpt:6",
                "name": "chatbot",
                "parameters": null
              }
            }
          ],
          "Messages": [
            {
              "role": "system",
              "content": [
                {
                  "text": "Call chatbot"
                }
              ]
            },
            {
              "role": "user",
              "content": [
                {
                  "text": "Hello"
                }
              ]
            },
            {
              "role": "assistant",
              "content": [
                {
                  "toolCall": {
                    "index": 0,
                    "id": "call_1",
                    "function": {
                      "name": "chatbot"
                    }
                  }
                }
              ]
            }
          ],
          "MaxTokens": 0,
          "Temperature": null,
          "JSONResponse": false,
          "Grammar": "",
          "Cache": null
        },
        "pending": {
          "call_1": {
            "index": 0,
            "id": "call_1",
            "function": {
              "name": "chatbot"
            }
          }
        }
      },
      "calls": {
        "call_1": {
          "toolID": "testdata/TestSubChat/test.gpt:6"
        }
      }
    },
    "subCalls": [
      {
        "toolId": "testdata/TestSubChat/test.gpt:6",
        "callId": "call_1",
        "state": {
          "continuation": {
            "state": {
              "completion": {
                "Model": "gpt-4-turbo-preview",
                "InternalSystemPrompt": false,
                "Tools": null,
                "Messages": [
                  {
                    "role": "system",
                    "content": [
                      {
                        "text": "This is a chatbot"
                      }
                    ]
                  },
                  {
                    "role": "assistant",
                    "content": [
                      {
                        "text": "Assistant 1"
                      }
                    ]
                  }
                ],
                "MaxTokens": 0,
                "Temperature": null,
                "JSONResponse": false,
                "Grammar": "",
                "Cache": null
              }
            },
            "result": "Assistant 1"
          },
          "continuationToolID": "testdata/TestSubChat/test.gpt:6"
        }
      }
    ],
    "subCallID": "call_1"
  }
}`).Equal(t, toJSONString(t, resp))

	resp, err = r.Chat(context.Background(), resp.State, prg, os.Environ(), "User 1")
	require.NoError(t, err)

	autogold.Expect(`{
  "done": false,
  "content": "Assistant 2",
  "toolID": "testdata/TestSubChat/test.gpt:6",
  "state": {
    "continuation": {
      "state": {
        "input": "Hello",
        "completion": {
          "Model": "gpt-4-turbo-preview",
          "InternalSystemPrompt": null,
          "Tools": [
            {
              "function": {
                "toolID": "testdata/TestSubChat/test.gpt:6",
                "name": "chatbot",
                "parameters": null
              }
            }
          ],
          "Messages": [
            {
              "role": "system",
              "content": [
                {
                  "text": "Call chatbot"
                }
              ]
            },
            {
              "role": "user",
              "content": [
                {
                  "text": "Hello"
                }
              ]
            },
            {
              "role": "assistant",
              "content": [
                {
                  "toolCall": {
                    "index": 0,
                    "id": "call_1",
                    "function": {
                      "name": "chatbot"
                    }
                  }
                }
              ]
            }
          ],
          "MaxTokens": 0,
          "Temperature": null,
          "JSONResponse": false,
          "Grammar": "",
          "Cache": null
        },
        "pending": {
          "call_1": {
            "index": 0,
            "id": "call_1",
            "function": {
              "name": "chatbot"
            }
          }
        }
      },
      "calls": {
        "call_1": {
          "toolID": "testdata/TestSubChat/test.gpt:6"
        }
      }
    },
    "subCalls": [
      {
        "toolId": "testdata/TestSubChat/test.gpt:6",
        "callId": "call_1",
        "state": {
          "continuation": {
            "state": {
              "completion": {
                "Model": "gpt-4-turbo-preview",
                "InternalSystemPrompt": false,
                "Tools": null,
                "Messages": [
                  {
                    "role": "system",
                    "content": [
                      {
                        "text": "This is a chatbot"
                      }
                    ]
                  },
                  {
                    "role": "assistant",
                    "content": [
                      {
                        "text": "Assistant 1"
                      }
                    ]
                  },
                  {
                    "role": "user",
                    "content": [
                      {
                        "text": "User 1"
                      }
                    ]
                  },
                  {
                    "role": "assistant",
                    "content": [
                      {
                        "text": "Assistant 2"
                      }
                    ]
                  }
                ],
                "MaxTokens": 0,
                "Temperature": null,
                "JSONResponse": false,
                "Grammar": "",
                "Cache": null
              }
            },
            "result": "Assistant 2"
          },
          "continuationToolID": "testdata/TestSubChat/test.gpt:6"
        }
      }
    ],
    "subCallID": "call_1"
  }
}`).Equal(t, toJSONString(t, resp))
}

func TestChat(t *testing.T) {
	r := tester.NewRunner(t)
	r.RespondWith(tester.Result{
		Text: "Assistant 1",
	}, tester.Result{
		Text: "Assistant 2",
	})

	prg, err := r.Load("")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, os.Environ(), "Hello")
	require.NoError(t, err)

	autogold.Expect(`{
  "done": false,
  "content": "Assistant 1",
  "toolID": "testdata/TestChat/test.gpt:1",
  "state": {
    "continuation": {
      "state": {
        "input": "Hello",
        "completion": {
          "Model": "gpt-4-turbo-preview",
          "InternalSystemPrompt": false,
          "Tools": null,
          "Messages": [
            {
              "role": "system",
              "content": [
                {
                  "text": "This is a chatbot"
                }
              ]
            },
            {
              "role": "user",
              "content": [
                {
                  "text": "Hello"
                }
              ]
            },
            {
              "role": "assistant",
              "content": [
                {
                  "text": "Assistant 1"
                }
              ]
            }
          ],
          "MaxTokens": 0,
          "Temperature": null,
          "JSONResponse": false,
          "Grammar": "",
          "Cache": null
        }
      },
      "result": "Assistant 1"
    },
    "continuationToolID": "testdata/TestChat/test.gpt:1"
  }
}`).Equal(t, toJSONString(t, resp))

	resp, err = r.Chat(context.Background(), resp.State, prg, os.Environ(), "User 1")
	require.NoError(t, err)

	autogold.Expect(`{
  "done": false,
  "content": "Assistant 2",
  "toolID": "testdata/TestChat/test.gpt:1",
  "state": {
    "continuation": {
      "state": {
        "input": "Hello",
        "completion": {
          "Model": "gpt-4-turbo-preview",
          "InternalSystemPrompt": false,
          "Tools": null,
          "Messages": [
            {
              "role": "system",
              "content": [
                {
                  "text": "This is a chatbot"
                }
              ]
            },
            {
              "role": "user",
              "content": [
                {
                  "text": "Hello"
                }
              ]
            },
            {
              "role": "assistant",
              "content": [
                {
                  "text": "Assistant 1"
                }
              ]
            },
            {
              "role": "user",
              "content": [
                {
                  "text": "User 1"
                }
              ]
            },
            {
              "role": "assistant",
              "content": [
                {
                  "text": "Assistant 2"
                }
              ]
            }
          ],
          "MaxTokens": 0,
          "Temperature": null,
          "JSONResponse": false,
          "Grammar": "",
          "Cache": null
        }
      },
      "result": "Assistant 2"
    },
    "continuationToolID": "testdata/TestChat/test.gpt:1"
  }
}`).Equal(t, toJSONString(t, resp))
}

func TestChatRunError(t *testing.T) {
	r := tester.NewRunner(t)
	_, err := r.Run("", "")
	autogold.Expect(`"{\n  \"continuation\": {\n    \"state\": {\n      \"completion\": {\n        \"Model\": \"gpt-4-turbo-preview\",\n        \"InternalSystemPrompt\": false,\n        \"Tools\": null,\n        \"Messages\": [\n          {\n            \"role\": \"system\",\n            \"content\": [\n              {\n                \"text\": \"This is a chatbot\"\n              }\n            ]\n          },\n          {\n            \"role\": \"assistant\",\n            \"content\": [\n              {\n                \"text\": \"TEST RESULT CALL: 1\"\n              }\n            ]\n          }\n        ],\n        \"MaxTokens\": 0,\n        \"Temperature\": null,\n        \"JSONResponse\": false,\n        \"Grammar\": \"\",\n        \"Cache\": null\n      }\n    },\n    \"result\": \"TEST RESULT CALL: 1\"\n  },\n  \"continuationToolID\": \"testdata/TestChatRunError/test.gpt:1\"\n}"`).Equal(t, toJSONString(t, toJSONString(t, err.(*runner.ErrContinuation).State)))
}

func TestExportContext(t *testing.T) {
	runner := tester.NewRunner(t)
	x := runner.RunDefault()
	assert.Equal(t, "TEST RESULT CALL: 1", x)
}

func TestContext(t *testing.T) {
	runner := tester.NewRunner(t)
	x := runner.RunDefault()
	assert.Equal(t, "TEST RESULT CALL: 1", x)
}

func TestContextArg(t *testing.T) {
	runner := tester.NewRunner(t)
	x, err := runner.Run("", `{
"file": "foo.db"
}`)
	require.NoError(t, err)
	assert.Equal(t, "TEST RESULT CALL: 1", x)
}

func TestCwd(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: types.ToolNormalizer("./subtool/test.gpt"),
		},
	})
	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "local",
		},
	})
	x := runner.RunDefault()
	assert.Equal(t, "TEST RESULT CALL: 3", x)
}

func TestExport(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "transient",
		},
	})
	x, err := runner.Run("parent.gpt", "")
	require.NoError(t, err)
	assert.Equal(t, "TEST RESULT CALL: 3", x)
}

func TestShlex(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "transient",
		},
	})
	x, err := runner.Run("", `{
"args":" x y  z  "
}`)
	require.NoError(t, err)
	// The fact the white space is all perfect means that the string didn't get passed as one arg but multiple
	assert.Equal(t, "x y z\n", x)
}
