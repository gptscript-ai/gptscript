package types

import (
	"testing"

	"github.com/hexops/autogold/v2"
)

func TestToolDef_Print(t *testing.T) {
	tool := ToolDef{
		Parameters: Parameters{
			Name:                "Tool Sample",
			Description:         "This is a sample tool",
			MaxTokens:           1024,
			ModelName:           "ModelSample",
			ModelProvider:       true,
			JSONResponse:        true,
			Chat:                true,
			Temperature:         float32Ptr(0.8),
			Cache:               boolPtr(true),
			InternalPrompt:      boolPtr(true),
			Arguments:           ObjectSchema("arg1", "desc1", "arg2", "desc2"),
			Tools:               []string{"Tool1", "Tool2"},
			GlobalTools:         []string{"GlobalTool1", "GlobalTool2"},
			GlobalModelName:     "GlobalModelSample",
			Context:             []string{"Context1", "Context2"},
			ExportContext:       []string{"ExportContext1", "ExportContext2"},
			Export:              []string{"Export1", "Export2"},
			Agents:              []string{"Agent1", "Agent2"},
			Credentials:         []string{"Credential1", "Credential2"},
			Blocking:            true,
			InputFilters:        []string{"Filter1", "Filter2"},
			ExportInputFilters:  []string{"SharedFilter1", "SharedFilter2"},
			OutputFilters:       []string{"Filter1", "Filter2"},
			ExportOutputFilters: []string{"SharedFilter1", "SharedFilter2"},
			ExportCredentials:   []string{"ExportCredential1", "ExportCredential2"},
			Type:                ToolTypeContext,
		},
		MetaData: map[string]string{
			"package.json": `{
// blah blah some ugly JSON
}
`,
			"requirements.txt": `requests=5`,
		},
		Instructions: "This is a sample instruction",
	}

	autogold.Expect(`Global Model Name: GlobalModelSample
Global Tools: GlobalTool1, GlobalTool2
Name: Tool Sample
Description: This is a sample tool
Type: Context
Agents: Agent1, Agent2
Tools: Tool1, Tool2
Share Tools: Export1, Export2
Context: Context1, Context2
Share Context: ExportContext1, ExportContext2
Input Filters: Filter1, Filter2
Share Input Filters: SharedFilter1, SharedFilter2
Output Filters: Filter1, Filter2
Share Output Filters: SharedFilter1, SharedFilter2
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
Share Credential: ExportCredential1
Share Credential: ExportCredential2
Chat: true
Meta Data: requirements.txt: requests=5

This is a sample instruction
---
!metadata:Tool Sample:package.json
{
// blah blah some ugly JSON
}

`).Equal(t, tool.Print())
}

// float32Ptr is used to return a pointer to a given float32 value
func float32Ptr(f float32) *float32 {
	return &f
}

// boolPtr is used to return a pointer to a given bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestSplitArg(t *testing.T) {
	prefix, arg := SplitArg("")
	autogold.Expect([]string{"", ""}).Equal(t, []string{prefix, arg})

	prefix, arg = SplitArg("toolName")
	autogold.Expect([]string{"toolName", ""}).Equal(t, []string{prefix, arg})

	prefix, arg = SplitArg("toolName as myAlias")
	autogold.Expect([]string{"toolName", "as myAlias"}).Equal(t, []string{prefix, arg})

	prefix, arg = SplitArg("toolName with value1 as arg1 and value2 as arg2")
	autogold.Expect([]string{"toolName", "value1 as arg1 and value2 as arg2"}).Equal(t, []string{prefix, arg})

	prefix, arg = SplitArg("toolName as myAlias with value1 as arg1 and value2 as arg2")
	autogold.Expect([]string{"toolName", "value1 as arg1 and value2 as arg2"}).Equal(t, []string{prefix, arg})

	prefix, arg = SplitArg("toolName with value1 as arg1 and value2 as arg2 as myAlias")
	autogold.Expect([]string{"toolName", "value1 as arg1 and value2 as arg2 as myAlias"}).Equal(t, []string{prefix, arg})
}
