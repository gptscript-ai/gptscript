package builtin

import (
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	defaultModel = openai.DefaultModel
)

func GetDefaultModel() string {
	return defaultModel
}

func SetDefaultModel(model string) {
	defaultModel = model
}

func SetDefaults(tool types.Tool) types.Tool {
	if tool.ModelName == "" {
		tool.ModelName = GetDefaultModel()
	}
	return tool
}
