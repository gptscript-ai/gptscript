package tests

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func TestContextWithAsterick(t *testing.T) {
	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(context.Background(), `
chat: true
context: foo with *

Say hi

---
name: foo

#!/bin/bash

echo This is the input: ${GPTSCRIPT_INPUT}
`, "")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, nil, "input 1", runner.RunOptions{})
	r.AssertStep(t, resp, err)

	resp, err = r.Chat(context.Background(), resp.State, prg, nil, "input 2", runner.RunOptions{})
	r.AssertStep(t, resp, err)
}

func TestContextShareBug(t *testing.T) {
	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(context.Background(), `
chat: true
tools: sharecontext

Say hi

---
name: sharecontext
share context: realcontext
---
name: realcontext

#!sys.echo

Yo dawg`, "")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, nil, "input 1", runner.RunOptions{})
	r.AssertStep(t, resp, err)
}

func TestInputFilterMoreArgs(t *testing.T) {
	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(context.Background(), `
chat: true
inputfilters: stuff

Say hi

---
name: stuff
params: foo: bar
params: input: baz

#!/bin/bash
echo ${FOO}:${INPUT}
`, "")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, nil, `{"foo":"123"}`, runner.RunOptions{})
	r.AssertStep(t, resp, err)
	resp, err = r.Chat(context.Background(), nil, prg, nil, `"foo":"123"}`, runner.RunOptions{})
	r.AssertStep(t, resp, err)
}

func TestShareCreds(t *testing.T) {
	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(context.Background(), `
creds: foo

#!/bin/bash
echo $CRED
echo $CRED2

---
name: foo
share credentials: bar

---
name: bar
share credentials: baz

#!/bin/bash
echo '{"env": {"CRED": "that worked"}}'

---
name: baz

#!/bin/bash
echo '{"env": {"CRED2": "that also worked"}}'
`, "")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, nil, "", runner.RunOptions{})
	r.AssertStep(t, resp, err)
}

func TestFilterArgs(t *testing.T) {
	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(context.Background(), `
inputfilters: input with ${Foo}
inputfilters: input with foo
inputfilters: input with *
outputfilters: output with *
outputfilters: output with foo
outputfilters: output with ${Foo}
params: Foo: a description

#!/bin/bash
echo ${FOO}

---
name: input
params: notfoo: a description

#!/bin/bash
echo "${GPTSCRIPT_INPUT}"

---
name: output
params: notfoo: a description

#!/bin/bash
echo "${GPTSCRIPT_INPUT}"
`, "")
	require.NoError(t, err)

	resp, err := r.Chat(context.Background(), nil, prg, nil, `{"foo":"baz", "start": true}`, runner.RunOptions{})
	r.AssertStep(t, resp, err)

	data := map[string]any{}
	err = json.Unmarshal([]byte(resp.Content), &data)
	require.NoError(t, err)

	autogold.Expect(map[string]interface{}{
		"chat":         false,
		"continuation": false,
		"notfoo":       "baz",
		"output": `{"chat":false,"continuation":false,"notfoo":"foo","output":"{\"chat\":false,\"continuation\":false,\"foo\":\"baz\",\"input\":\"{\\\"foo\\\":\\\"baz\\\",\\\"input\\\":\\\"{\\\\\\\"foo\\\\\\\":\\\\\\\"baz\\\\\\\", \\\\\\\"start\\\\\\\": true}\\\",\\\"notfoo\\\":\\\"baz\\\",\\\"start\\\":true}\\n\",\"notfoo\":\"foo\",\"output\":\"baz\\n\",\"start\":true}\n"}
`,
	}).Equal(t, data)

	val := data["output"].(string)
	data = map[string]any{}
	err = json.Unmarshal([]byte(val), &data)
	require.NoError(t, err)
	autogold.Expect(map[string]interface{}{
		"chat":         false,
		"continuation": false,
		"notfoo":       "foo",
		"output": `{"chat":false,"continuation":false,"foo":"baz","input":"{\"foo\":\"baz\",\"input\":\"{\\\"foo\\\":\\\"baz\\\", \\\"start\\\": true}\",\"notfoo\":\"baz\",\"start\":true}\n","notfoo":"foo","output":"baz\n","start":true}
`,
	}).Equal(t, data)

	val = data["output"].(string)
	data = map[string]any{}
	err = json.Unmarshal([]byte(val), &data)
	require.NoError(t, err)
	autogold.Expect(map[string]interface{}{
		"chat":         false,
		"continuation": false,
		"foo":          "baz", "input": `{"foo":"baz","input":"{\"foo\":\"baz\", \"start\": true}","notfoo":"baz","start":true}
`,
		"notfoo": "foo",
		"output": "baz\n",
		"start":  true,
	}).Equal(t, data)

	val = data["input"].(string)
	data = map[string]any{}
	err = json.Unmarshal([]byte(val), &data)
	require.NoError(t, err)
	autogold.Expect(map[string]interface{}{
		"foo":    "baz",
		"input":  `{"foo":"baz", "start": true}`,
		"notfoo": "baz",
		"start":  true,
	}).Equal(t, data)

	val = data["input"].(string)
	data = map[string]any{}
	err = json.Unmarshal([]byte(val), &data)
	require.NoError(t, err)
	autogold.Expect(map[string]interface{}{"foo": "baz", "start": true}).Equal(t, data)
}

func TestMCPLoad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	r := tester.NewRunner(t)
	prg, err := loader.ProgramFromSource(t.Context(), `
name: mcp

#!mcp

{
	"mcpServers": {
	  "sqlite": {
		"command": "docker",
		"args": [
		  "run",
		  "--rm",
		  "-i",
		  "-v",
		  "mcp-test:/mcp",
		  "mcp/sqlite@sha256:007ccae941a6f6db15b26ee41d92edda50ce157176d9273449e8b3f51d979c70",
		  "--db-path",
		  "/mcp/test.db"
		]
	  }
	}
}
`, "")
	require.NoError(t, err)

	autogold.Expect(types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name:        "mcp",
				Description: "sqlite",
				ModelName:   "gpt-4o",
				Export: []string{
					"read_query",
					"write_query",
					"create_table",
					"list_tables",
					"describe_table",
					"append_insight",
				},
			},
			MetaData: map[string]string{"bundle": "true"},
		},
		ID: "inline:mcp",
		ToolMapping: map[string][]types.ToolReference{
			"append_insight": {{
				Reference: "append_insight",
				ToolID:    "inline:append_insight",
			}},
			"create_table": {{
				Reference: "create_table",
				ToolID:    "inline:create_table",
			}},
			"describe_table": {{
				Reference: "describe_table",
				ToolID:    "inline:describe_table",
			}},
			"list_tables": {{
				Reference: "list_tables",
				ToolID:    "inline:list_tables",
			}},
			"read_query": {{
				Reference: "read_query",
				ToolID:    "inline:read_query",
			}},
			"write_query": {{
				Reference: "write_query",
				ToolID:    "inline:write_query",
			}},
		},
		LocalTools: map[string]string{
			"append_insight": "inline:append_insight",
			"create_table":   "inline:create_table",
			"describe_table": "inline:describe_table",
			"list_tables":    "inline:list_tables",
			"mcp":            "inline:mcp",
			"read_query":     "inline:read_query",
			"write_query":    "inline:write_query",
		},
		Source:     types.ToolSource{Location: "inline"},
		WorkingDir: ".",
	}).Equal(t, prg.ToolSet[prg.EntryToolID])
	autogold.Expect(7).Equal(t, len(prg.ToolSet[prg.EntryToolID].LocalTools))
	data, _ := json.MarshalIndent(prg.ToolSet, "", "  ")
	autogold.Expect(`{
  "inline:append_insight": {
    "name": "append_insight",
    "description": "Add a business insight to the memo",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "properties": {
        "insight": {
          "description": "Business insight discovered from data analysis",
          "type": "string"
        }
      },
      "required": [
        "insight"
      ],
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.append_insight e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:append_insight",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:create_table": {
    "name": "create_table",
    "description": "Create a new table in the SQLite database",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "properties": {
        "query": {
          "description": "CREATE TABLE SQL statement",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.create_table e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:create_table",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:describe_table": {
    "name": "describe_table",
    "description": "Get the schema information for a specific table",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "properties": {
        "table_name": {
          "description": "Name of the table to describe",
          "type": "string"
        }
      },
      "required": [
        "table_name"
      ],
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.describe_table e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:describe_table",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:list_tables": {
    "name": "list_tables",
    "description": "List all tables in the SQLite database",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.list_tables e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:list_tables",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:mcp": {
    "name": "mcp",
    "description": "sqlite",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "export": [
      "read_query",
      "write_query",
      "create_table",
      "list_tables",
      "describe_table",
      "append_insight"
    ],
    "metaData": {
      "bundle": "true"
    },
    "id": "inline:mcp",
    "toolMapping": {
      "append_insight": [
        {
          "reference": "append_insight",
          "toolID": "inline:append_insight"
        }
      ],
      "create_table": [
        {
          "reference": "create_table",
          "toolID": "inline:create_table"
        }
      ],
      "describe_table": [
        {
          "reference": "describe_table",
          "toolID": "inline:describe_table"
        }
      ],
      "list_tables": [
        {
          "reference": "list_tables",
          "toolID": "inline:list_tables"
        }
      ],
      "read_query": [
        {
          "reference": "read_query",
          "toolID": "inline:read_query"
        }
      ],
      "write_query": [
        {
          "reference": "write_query",
          "toolID": "inline:write_query"
        }
      ]
    },
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:read_query": {
    "name": "read_query",
    "description": "Execute a SELECT query on the SQLite database",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "properties": {
        "query": {
          "description": "SELECT SQL query to execute",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.read_query e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:read_query",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  },
  "inline:write_query": {
    "name": "write_query",
    "description": "Execute an INSERT, UPDATE, or DELETE query on the SQLite database",
    "modelName": "gpt-4o",
    "internalPrompt": null,
    "arguments": {
      "properties": {
        "query": {
          "description": "SQL query to execute",
          "type": "string"
        }
      },
      "required": [
        "query"
      ],
      "type": "object"
    },
    "instructions": "#!sys.mcp.invoke.write_query e592cc0c9483290685611ba70bd8595829cc794f7eae0419eabb3388bf0d3529",
    "id": "inline:write_query",
    "localTools": {
      "append_insight": "inline:append_insight",
      "create_table": "inline:create_table",
      "describe_table": "inline:describe_table",
      "list_tables": "inline:list_tables",
      "mcp": "inline:mcp",
      "read_query": "inline:read_query",
      "write_query": "inline:write_query"
    },
    "source": {
      "location": "inline"
    },
    "workingDir": "."
  }
}`).Equal(t, string(data))

	prg.EntryToolID = prg.ToolSet[prg.EntryToolID].LocalTools["read_query"]
	resp, err := r.Chat(context.Background(), nil, prg, nil, `{"query": "SELECT 1"}`, runner.RunOptions{})
	r.AssertStep(t, resp, err)
}
