package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func toString(obj any) string {
	s, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(s)
}

func TestLocalRemote(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
	}))
	defer s.Close()
	dir, err := os.MkdirTemp("", "gptscript-test")
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "chatbot.gpt"), []byte(fmt.Sprintf(`
Chat: true
Name: chatbot
Context: context.gpt
Tools: http://%s/swagger.json

THis is a tool, say hi
`, s.Listener.Addr().String())), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "context.gpt"), []byte(`
#!sys.echo

Stuff
`), 0644)
	require.NoError(t, err)

	_, err = Program(context.Background(), filepath.Join(dir, "chatbot.gpt"), "")
	require.NoError(t, err)
}

func TestIsOpenAPI(t *testing.T) {
	datav2, err := os.ReadFile("testdata/openapi_v2.yaml")
	require.NoError(t, err)
	v := isOpenAPI(datav2)
	require.Equal(t, 2, v, "(yaml) expected openapi v2")

	datav2, err = os.ReadFile("testdata/openapi_v2.json")
	require.NoError(t, err)
	v = isOpenAPI(datav2)
	require.Equal(t, 2, v, "(json) expected openapi v2")

	datav3, err := os.ReadFile("testdata/openapi_v3.yaml")
	require.NoError(t, err)
	v = isOpenAPI(datav3)
	require.Equal(t, 3, v, "(json) expected openapi v3")
}

func TestLoadOpenAPI(t *testing.T) {
	numOpenAPITools := func(set types.ToolSet) int {
		num := 0
		for _, v := range set {
			if v.IsOpenAPI() {
				num++
			}
		}
		return num
	}

	prgv3 := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav3, err := os.ReadFile("testdata/openapi_v3.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv3, &source{Content: datav3}, "")
	require.NoError(t, err, "failed to read openapi v3")
	require.Equal(t, 3, numOpenAPITools(prgv3.ToolSet), "expected 3 openapi tools")

	prgv2json := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav2, err := os.ReadFile("testdata/openapi_v2.json")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv2json, &source{Content: datav2}, "")
	require.NoError(t, err, "failed to read openapi v2")
	require.Equal(t, 3, numOpenAPITools(prgv2json.ToolSet), "expected 3 openapi tools")

	prgv2yaml := types.Program{
		ToolSet: types.ToolSet{},
	}
	datav2, err = os.ReadFile("testdata/openapi_v2.yaml")
	require.NoError(t, err)
	_, err = readTool(context.Background(), nil, &prgv2yaml, &source{Content: datav2}, "")
	require.NoError(t, err, "failed to read openapi v2 (yaml)")
	require.Equal(t, 3, numOpenAPITools(prgv2yaml.ToolSet), "expected 3 openapi tools")

	require.EqualValuesf(t, prgv2json.ToolSet, prgv2yaml.ToolSet, "expected same toolset for openapi v2 json and yaml")
}

func TestHelloWorld(t *testing.T) {
	prg, err := Program(context.Background(),
		"https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt",
		"")
	require.NoError(t, err)
	autogold.Expect(`{
  "name": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt",
  "entryToolId": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:",
  "toolSet": {
    "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:": {
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "instructions": "Say hello world",
      "id": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:",
      "localTools": {
        "": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:"
      },
      "source": {
        "location": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt",
        "lineNo": 1
      },
      "workingDir": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example"
    },
    "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:": {
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "tools": [
        "../bob.gpt"
      ],
      "instructions": "call bob",
      "id": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:",
      "toolMapping": {
        "../bob.gpt": [
          {
            "reference": "../bob.gpt",
            "toolID": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:"
          }
        ]
      },
      "localTools": {
        "": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:"
      },
      "source": {
        "location": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt",
        "lineNo": 1
      },
      "workingDir": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub"
    }
  }
}`).Equal(t, toString(prg))

	prg, err = Program(context.Background(), "https://get.gptscript.ai/echo.gpt", "")
	require.NoError(t, err)

	autogold.Expect(`{
  "name": "https://get.gptscript.ai/echo.gpt",
  "entryToolId": "https://get.gptscript.ai/echo.gpt:",
  "toolSet": {
    "https://get.gptscript.ai/echo.gpt:": {
      "description": "Returns back the input of the script",
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "arguments": {
        "properties": {
          "input": {
            "description": "Any string",
            "type": "string"
          }
        },
        "type": "object"
      },
      "instructions": "echo \"${input}\"",
      "id": "https://get.gptscript.ai/echo.gpt:",
      "localTools": {
        "": "https://get.gptscript.ai/echo.gpt:"
      },
      "source": {
        "location": "https://get.gptscript.ai/echo.gpt",
        "lineNo": 1
      },
      "workingDir": "https://get.gptscript.ai/"
    }
  }
}`).Equal(t, toString(prg))
}

func TestDefault(t *testing.T) {
	prg, err := Program(context.Background(), "./testdata/tool", "")
	require.NoError(t, err)
	autogold.Expect(`{
  "name": "./testdata/tool",
  "entryToolId": "testdata/tool/tool.gpt:tool",
  "toolSet": {
    "testdata/tool/tool.gpt:tool": {
      "name": "tool",
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "instructions": "a tool",
      "id": "testdata/tool/tool.gpt:tool",
      "localTools": {
        "tool": "testdata/tool/tool.gpt:tool"
      },
      "source": {
        "location": "testdata/tool/tool.gpt",
        "lineNo": 1
      },
      "workingDir": "testdata/tool"
    }
  }
}`).Equal(t, toString(prg))

	prg, err = Program(context.Background(), "./testdata/agent", "")
	require.NoError(t, err)
	autogold.Expect(`{
  "name": "./testdata/agent",
  "entryToolId": "testdata/agent/agent.gpt:agent",
  "toolSet": {
    "testdata/agent/agent.gpt:agent": {
      "name": "agent",
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "instructions": "an agent",
      "id": "testdata/agent/agent.gpt:agent",
      "localTools": {
        "agent": "testdata/agent/agent.gpt:agent"
      },
      "source": {
        "location": "testdata/agent/agent.gpt",
        "lineNo": 1
      },
      "workingDir": "testdata/agent"
    }
  }
}`).Equal(t, toString(prg))

	prg, err = Program(context.Background(), "./testdata/bothtoolagent", "")
	require.NoError(t, err)
	autogold.Expect(`{
  "name": "./testdata/bothtoolagent",
  "entryToolId": "testdata/bothtoolagent/agent.gpt:agent",
  "toolSet": {
    "testdata/bothtoolagent/agent.gpt:agent": {
      "name": "agent",
      "modelName": "gpt-4o",
      "internalPrompt": null,
      "instructions": "an agent",
      "id": "testdata/bothtoolagent/agent.gpt:agent",
      "localTools": {
        "agent": "testdata/bothtoolagent/agent.gpt:agent"
      },
      "source": {
        "location": "testdata/bothtoolagent/agent.gpt",
        "lineNo": 1
      },
      "workingDir": "testdata/bothtoolagent"
    }
  }
}`).Equal(t, toString(prg))
}
