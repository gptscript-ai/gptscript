package tests

import (
	"context"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
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
