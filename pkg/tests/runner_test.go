package tests

import (
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestCwd(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "subtool/test",
		},
	})
	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "local",
		},
	})
	x := runner.RunDefault()
	assert.Equal(t, "TEST RESULT CALL: 3", x)
}
