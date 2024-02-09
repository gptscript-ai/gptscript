package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

const (
	examplePath = "../../examples"
)

func TestExamples(t *testing.T) {
	RequireOpenAPIKey(t)

	tests := []string{
		"fac.gpt",
		"helloworld.gpt",
	}
	for _, entry := range tests {
		t.Run(entry, func(t *testing.T) {
			r, err := runner.New()
			require.NoError(t, err)

			prg, err := loader.Program(context.Background(), filepath.Join(examplePath, entry), "")
			require.NoError(t, err)

			output, err := r.Run(context.Background(), prg, os.Environ(), "")
			require.NoError(t, err)

			autogold.ExpectFile(t, output)
		})
	}
}

func TestEcho(t *testing.T) {
	RequireOpenAPIKey(t)

	r, err := runner.New()
	require.NoError(t, err)

	prg, err := loader.Program(context.Background(), filepath.Join(examplePath, "echo.gpt"), "")
	require.NoError(t, err)

	output, err := r.Run(context.Background(), prg, os.Environ(), "this is a test")
	require.NoError(t, err)

	autogold.ExpectFile(t, output)
}

func RequireOpenAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip()
	}
}
