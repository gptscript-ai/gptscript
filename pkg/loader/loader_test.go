package loader

import (
	"context"
	"encoding/json"
	"testing"

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

func TestHelloWorld(t *testing.T) {
	prg, err := Program(context.Background(),
		"https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt",
		"")
	require.NoError(t, err)
	autogold.Expect(`{
  "name": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt",
  "entryToolId": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:1",
  "toolSet": {
    "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:1": {
      "modelName": "gpt-4-turbo-preview",
      "internalPrompt": null,
      "instructions": "Say hello world",
      "id": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:1",
      "localTools": {
        "": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:1"
      },
      "source": {
        "location": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt",
        "lineNo": 1
      },
      "workingDir": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example"
    },
    "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:1": {
      "modelName": "gpt-4-turbo-preview",
      "internalPrompt": null,
      "tools": [
        "../bob.gpt"
      ],
      "instructions": "call bob",
      "id": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:1",
      "toolMapping": {
        "../bob.gpt": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/bob.gpt:1"
      },
      "localTools": {
        "": "https://raw.githubusercontent.com/ibuildthecloud/test/bafe5a62174e8a0ea162277dcfe3a2ddb7eea928/example/sub/tool.gpt:1"
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
  "entryToolId": "https://get.gptscript.ai/echo.gpt:1",
  "toolSet": {
    "https://get.gptscript.ai/echo.gpt:1": {
      "description": "Returns back the input of the script",
      "modelName": "gpt-4-turbo-preview",
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
      "id": "https://get.gptscript.ai/echo.gpt:1",
      "localTools": {
        "": "https://get.gptscript.ai/echo.gpt:1"
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
