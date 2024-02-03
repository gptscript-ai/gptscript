package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/acorn-io/gptscript/pkg/loader"
	"github.com/acorn-io/gptscript/pkg/runner"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

const (
	examplePath = "../../examples"
)

func TestExamples(t *testing.T) {
	l, err := os.ReadDir(examplePath)
	require.NoError(t, err)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range l {
		entry.Name()
		t.Run(entry.Name(), func(t *testing.T) {
			r, err := runner.New()
			require.NoError(t, err)

			prg, err := loader.Program(context.Background(), filepath.Join(examplePath, entry.Name()), "")
			require.NoError(t, err)

			output, err := r.Run(context.Background(), prg, os.Environ(), "")
			require.NoError(t, err)

			autogold.ExpectFile(t, output)
		})
	}
}
