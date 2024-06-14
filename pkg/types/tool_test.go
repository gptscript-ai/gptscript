package types

import (
	"testing"

	"github.com/hexops/autogold/v2"
)

func TestToolDef_String(t *testing.T) {
	tool := ToolDef{
		Parameters: Parameters{
			Name:            "Tool Sample",
			Description:     "This is a sample tool",
			MaxTokens:       1024,
			ModelName:       "ModelSample",
			ModelProvider:   true,
			JSONResponse:    true,
			Chat:            true,
			Temperature:     float32Ptr(0.8),
			Cache:           boolPtr(true),
			InternalPrompt:  boolPtr(true),
			Arguments:       ObjectSchema("arg1", "desc1", "arg2", "desc2"),
			Tools:           []string{"Tool1", "Tool2"},
			GlobalTools:     []string{"GlobalTool1", "GlobalTool2"},
			GlobalModelName: "GlobalModelSample",
			Context:         []string{"Context1", "Context2"},
			ExportContext:   []string{"ExportContext1", "ExportContext2"},
			Export:          []string{"Export1", "Export2"},
			Agents:          []string{"Agent1", "Agent2"},
			Credentials:     []string{"Credential1", "Credential2"},
			Blocking:        true,
		},
		Instructions: "This is a sample instruction",
	}

	autogold.Expect(`Global Model Name: GlobalModelSample
Global Tools: GlobalTool1, GlobalTool2
Name: Tool Sample
Description: This is a sample tool
Agents: Agent1, Agent2
Tools: Tool1, Tool2
Share Tools: Export1, Export2
Share Context: ExportContext1, ExportContext2
Context: Context1, Context2
Max Tokens: 1024
Model: ModelSample
Model Provider: true
JSON Response: true
Temperature: 0.800000
Parameter: arg1: desc1
Parameter: arg2: desc2
Internal Prompt: true
Credential: Credential1
Credential: Credential2
Chat: true

This is a sample instruction
`).Equal(t, tool.String())
}

// float32Ptr is used to return a pointer to a given float32 value
func float32Ptr(f float32) *float32 {
	return &f
}

// boolPtr is used to return a pointer to a given bool value
func boolPtr(b bool) *bool {
	return &b
}
