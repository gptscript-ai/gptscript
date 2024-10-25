package repos

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/adrg/xdg"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/python"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCacheHome = lo.Must(xdg.CacheFile("gptscript-test-cache/runtime"))
)

func TestManager_GetContext(t *testing.T) {
	m := New(testCacheHome, "", &python.Runtime{
		Version: "3.11",
	})
	cwd, env, err := m.GetContext(context.Background(), types.Tool{
		Source: types.ToolSource{
			Repo: &types.Repo{
				VCS:      "git",
				Root:     "https://github.com/gptscript-ai/dalle-image-generation.git",
				Revision: "b9d9ed60c25da7c0e01d504a7219d1c6e460fe80",
			},
		},
	}, []string{"/usr/bin/env", "python3.11"}, os.Environ())
	require.NoError(t, err)
	assert.NotEqual(t, "", cwd)
	assert.True(t, len(env) > 0)
	fmt.Print(cwd)
	fmt.Print(env)
}
