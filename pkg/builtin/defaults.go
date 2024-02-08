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
	if tool.ModelName == "" {
		if tool.Vision {
			tool.ModelName = DefaultVisionModel
		} else {
			tool.ModelName = DefaultModel
		}
	}
	return tool
}
