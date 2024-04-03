package tests

import (
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/tests/tester"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCwd(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: types.ToolNormalizer("./subtool/test.gpt"),
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

func TestExport(t *testing.T) {
	runner := tester.NewRunner(t)

	runner.RespondWith(tester.Result{
		Func: types.CompletionFunctionCall{
			Name: "transient",
		},
	})
	x, err := runner.Run("parent.gpt", "")
	require.NoError(t, err)
	assert.Equal(t, "TEST RESULT CALL: 3", x)
}
