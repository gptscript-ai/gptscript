package loader

import (
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	DefaultModel       = openai.DefaultModel
	DefaultVisionModel = openai.DefaultVisionModel
	DefaultToolSchema  = types.JSONSchema{
		Property: types.Property{
			Type: "object",
		},
		Properties: map[string]types.Property{
			openai.DefaultPromptParameter: {
				Description: "Prompt to send to the tool or assistant. This may be instructions or question.",
				Type:        "string",
			},
		},
		Required: []string{openai.DefaultPromptParameter},
	}
)

func SetDefaults(tool types.Tool) types.Tool {
	if !tool.IsCommand() && tool.Arguments == nil {
		tool.Arguments = &DefaultToolSchema
	}
	if tool.ModelName == "" {
		if tool.Vision {
			tool.ModelName = DefaultVisionModel
		} else {
			tool.ModelName = DefaultModel
		}
	}
	return tool
}
