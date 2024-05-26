package loader

import (
	"context"
	"encoding/json"
	"os"
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
