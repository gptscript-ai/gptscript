package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

var (
	testCacheHome = lo.Must(xdg.CacheFile("gptscript-test-cache/repo"))
	testCommit    = "f9d0ca6559d0b7c78da7f413fc4faf87ae9b8919"
)

func TestFetch(t *testing.T) {
	err := fetch(context.Background(), testCacheHome,
		"https://github.com/gptscript-ai/dalle-image-generation.git",
		testCommit)
	require.NoError(t, err)
}

func TestCheckout(t *testing.T) {
	commitDir := filepath.Join(testCacheHome, "commits", testCommit)
	err := os.RemoveAll(commitDir)
	require.NoError(t, err)
	err = Checkout(context.Background(), testCacheHome,
		"https://github.com/gptscript-ai/dalle-image-generation.git",
		testCommit, commitDir)
	require.NoError(t, err)
}
