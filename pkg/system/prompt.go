package system

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
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

var DefaultToolSchema = openapi3.Schema{
	Type: &openapi3.Types{"object"},
	Properties: openapi3.Schemas{
		DefaultPromptParameter: &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Description: "Prompt to send to the tool. This may be an instruction or question.",
				Type:        &openapi3.Types{"string"},
			},
		},
	},
}

var DefaultChatSchema = openapi3.Schema{
	Type: &openapi3.Types{"object"},
	Properties: openapi3.Schemas{
		DefaultPromptParameter: &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Description: "Prompt to send to the assistant. This may be an instruction or question.",
				Type:        &openapi3.Types{"string"},
			},
		},
	},
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
