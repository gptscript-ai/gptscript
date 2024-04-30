package parser

import (
	"strings"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func TestParseGlobals(t *testing.T) {
	var input = `
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
		{ToolNode: &ToolNode{
			Tool: types.Tool{
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
				Source: types.ToolSource{LineNo: 1},
			},
		}},
		{ToolNode: &ToolNode{Tool: types.Tool{
			Parameters: types.Parameters{
				Name:      "bar",
				ModelName: "the model",
				Tools: []string{
					"bar",
					"foo",
				},
			},
			Source: types.ToolSource{LineNo: 5},
		}}},
	}}).Equal(t, out)
}

func TestParseSkip(t *testing.T) {
	var input = `
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
		{ToolNode: &ToolNode{
			Tool: types.Tool{
				Instructions: "first",
				Source: types.ToolSource{
					LineNo: 1,
				},
			},
		}},
		{ToolNode: &ToolNode{Tool: types.Tool{
			Parameters: types.Parameters{Name: "second"},
			Source:     types.ToolSource{LineNo: 4},
		}}},
		{TextNode: &TextNode{Text: "!third\n\nname: third\n"}},
		{ToolNode: &ToolNode{Tool: types.Tool{
			Parameters:   types.Parameters{Name: "fourth"},
			Instructions: "!forth dont skip",
			Source:       types.ToolSource{LineNo: 11},
		}}},
		{ToolNode: &ToolNode{Tool: types.Tool{
			Parameters:   types.Parameters{Name: "fifth"},
			Instructions: "#!ignore",
			Source:       types.ToolSource{LineNo: 14},
		}}},
		{TextNode: &TextNode{Text: `!skip
name: six

----
name: bad
 ---
name: bad
--
name: bad
---
name: bad
`}},
		{ToolNode: &ToolNode{Tool: types.Tool{
			Parameters: types.Parameters{
				Name: "seven",
			},
			Source: types.ToolSource{LineNo: 30},
		}}},
	}}).Equal(t, out)
}
