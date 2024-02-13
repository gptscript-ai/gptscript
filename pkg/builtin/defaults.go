package builtin

import (
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	DefaultModel       = openai.DefaultModel
	DefaultVisionModel = openai.DefaultVisionModel
)

func SetDefaults(tool types.Tool) types.Tool {
	if tool.Parameters.ModelName == "" {
		if tool.Parameters.Vision {
			tool.Parameters.ModelName = DefaultVisionModel
		} else {
			tool.Parameters.ModelName = DefaultModel
		}
	}
	return tool
}
