package tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
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

	resp, err := r.Chat(context.Background(), nil, prg, nil, "input 1")
	r.AssertStep(t, resp, err)

	resp, err = r.Chat(context.Background(), resp.State, prg, nil, "input 2")
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

	resp, err := r.Chat(context.Background(), nil, prg, nil, "input 1")
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

	resp, err := r.Chat(context.Background(), nil, prg, nil, `{"foo":"123"}`)
	r.AssertStep(t, resp, err)
	resp, err = r.Chat(context.Background(), nil, prg, nil, `"foo":"123"}`)
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

	resp, err := r.Chat(context.Background(), nil, prg, nil, "")
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

	resp, err := r.Chat(context.Background(), nil, prg, nil, `{"foo":"baz", "start": true}`)
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
