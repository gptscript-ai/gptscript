package builtin

import (
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	DefaultModel = openai.DefaultModel
)

func SetDefaults(tool types.Tool) types.Tool {
	if tool.Parameters.ModelName == "" {
		tool.Parameters.ModelName = DefaultModel
	}
	return tool
}
