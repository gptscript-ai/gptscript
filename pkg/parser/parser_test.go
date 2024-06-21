package parser

import (
	"strings"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func TestHashBang(t *testing.T) {
	autogold.Expect(true).Equal(t, isGPTScriptHashBang("#!/usr/bin/env gptscript"))
	autogold.Expect(true).Equal(t, isGPTScriptHashBang("#!/bin/env gptscript"))
	autogold.Expect(true).Equal(t, isGPTScriptHashBang("#!gptscript"))

	autogold.Expect(false).Equal(t, isGPTScriptHashBang("/usr/bin/env gptscript"))
	autogold.Expect(false).Equal(t, isGPTScriptHashBang("/bin/env gptscript"))
	autogold.Expect(false).Equal(t, isGPTScriptHashBang("gptscript"))

	autogold.Expect(false).Equal(t, isGPTScriptHashBang("#!/sr/bin/env gptscript"))
	autogold.Expect(false).Equal(t, isGPTScriptHashBang("#!/bin/env gpscript"))
	autogold.Expect(false).Equal(t, isGPTScriptHashBang("#!gptscrip"))
}

func TestParseGlobals(t *testing.T) {
	input := `
global tools: foo, bar
global model: the model
---
name: bar
tools: bar
`
	out, err := Parse(strings.NewReader(input), Options{
		AssignGlobals: true,
	})
	require.NoError(t, err)
	autogold.Expect(Document{Nodes: []Node{
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters: types.Parameters{
							ModelName: "the model",
							Tools: []string{
								"foo",
								"bar",
							},
							GlobalTools: []string{
								"foo",
								"bar",
							},
							GlobalModelName: "the model",
						},
					},
					Source: types.ToolSource{LineNo: 1},
				},
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters: types.Parameters{
							Name:      "bar",
							ModelName: "the model",
							Tools: []string{
								"bar",
								"foo",
							},
						},
					},
					Source: types.ToolSource{LineNo: 5},
				},
			},
		},
	}}).Equal(t, out)
}

func TestParseSkip(t *testing.T) {
	input := `
first
---
name: second
---

!third

name: third
---
name: fourth
!forth dont skip
---
name: fifth

#!ignore
---
!skip
name: six

----
name: bad
 ---
name: bad
--
name: bad
--- 
name: bad
---
name: seven
`
	out, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	autogold.Expect(Document{Nodes: []Node{
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Instructions: "first",
					},
					Source: types.ToolSource{
						LineNo: 1,
					},
				},
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters: types.Parameters{Name: "second"},
					},
					Source: types.ToolSource{LineNo: 4},
				},
			},
		},
		{
			TextNode: &TextNode{
				Text: "!third\n\nname: third\n",
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters:   types.Parameters{Name: "fourth"},
						Instructions: "!forth dont skip",
					},
					Source: types.ToolSource{LineNo: 11},
				},
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters:   types.Parameters{Name: "fifth"},
						Instructions: "#!ignore",
					},
					Source: types.ToolSource{LineNo: 14},
				},
			},
		},
		{
			TextNode: &TextNode{
				Text: `!skip
name: six

----
name: bad
 ---
name: bad
--
name: bad
---
name: bad
`,
			},
		},
		{
			ToolNode: &ToolNode{
				Tool: types.Tool{
					ToolDef: types.ToolDef{
						Parameters: types.Parameters{
							Name: "seven",
						},
					},
					Source: types.ToolSource{LineNo: 30},
				},
			},
		},
	}}).Equal(t, out)
}

func TestParseInput(t *testing.T) {
	input := `
input filters: input
share input filters: shared
`
	out, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	autogold.Expect(Document{Nodes: []Node{
		{ToolNode: &ToolNode{
			Tool: types.Tool{
				ToolDef: types.ToolDef{
					Parameters: types.Parameters{
						InputFilters: []string{
							"input",
						},
						ExportInputFilters: []string{"shared"},
					},
				},
				Source: types.ToolSource{LineNo: 1},
			},
		}},
	}}).Equal(t, out)
}

func TestParseOutput(t *testing.T) {
	output := `
output filters: output
share output filters: shared
`
	out, err := Parse(strings.NewReader(output))
	require.NoError(t, err)
	autogold.Expect(Document{Nodes: []Node{
		{ToolNode: &ToolNode{
			Tool: types.Tool{
				ToolDef: types.ToolDef{
					Parameters: types.Parameters{
						OutputFilters: []string{
							"output",
						},
						ExportOutputFilters: []string{"shared"},
					},
				},
				Source: types.ToolSource{LineNo: 1},
			},
		}},
	}}).Equal(t, out)
}
