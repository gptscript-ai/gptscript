package system

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

// Suffix is default suffix of gptscript files
const Suffix = ".gpt"

// InternalSystemPrompt is added to all threads. Changing this is very dangerous as it has a
// terrible global effect and changes the behavior of all scripts.
var InternalSystemPrompt = `
You are task oriented system.
You receive input from a user, process the input from the given instructions, and then output the result.
Your objective is to provide consistent and correct results.
You do not need to explain the steps taken, only provide the result to the given instructions.
You are referred to as a tool.
You don't move to the next step until you have a result.
`

// DefaultPromptParameter is used as the key in a json map to indication that we really wanted
// to just send pure text but the interface required JSON (as that is the fundamental interface of tools in OpenAI)
var DefaultPromptParameter = "defaultPromptParameter"

var DefaultToolSchema = types.JSONSchema{
	Property: types.Property{
		Type: "object",
	},
	Properties: map[string]types.Property{
		DefaultPromptParameter: {
			Description: "Prompt to send to the tool or assistant. This may be instructions or question.",
			Type:        "string",
		},
	},
	Required: []string{DefaultPromptParameter},
}

func init() {
	if p := os.Getenv("GPTSCRIPT_INTERNAL_SYSTEM_PROMPT"); p != "" {
		InternalSystemPrompt = p
	}
}

// IsDefaultPrompt Checks if the content is a json blob that has the defaultPromptParameter in it. If so
// it will extract out the value and return it. If not it will return the original content as is and false.
func IsDefaultPrompt(content string) (string, bool) {
	if strings.Contains(content, DefaultPromptParameter) && strings.HasPrefix(content, "{") {
		data := map[string]any{}
		if err := json.Unmarshal([]byte(content), &data); err == nil && len(data) == 1 {
			if v, _ := data[DefaultPromptParameter].(string); v != "" {
				return v, true
			}
		}
	}
	return content, false
}
