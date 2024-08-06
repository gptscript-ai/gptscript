package node

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

var (
	testCacheHome = lo.Must(xdg.CacheFile("gptscript-test-cache/runtime"))
)

func firstPath(s []string) string {
	_, p, _ := strings.Cut(s[0], "=")
	return strings.Split(p, string(os.PathListSeparator))[0]
}

func TestRuntime(t *testing.T) {
	r := Runtime{
		Version: "20",
	}

	s, err := r.Setup(context.Background(), types.Tool{}, testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(firstPath(s), "node.exe"))
	if errors.Is(err, fs.ErrNotExist) {
		_, err = os.Stat(filepath.Join(firstPath(s), "node"))
	}
	require.NoError(t, err)
}

func TestRuntime21(t *testing.T) {
	r := Runtime{
		Version: "21",
	}

	s, err := r.Setup(context.Background(), types.Tool{}, testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(firstPath(s), "node.exe"))
	if errors.Is(err, fs.ErrNotExist) {
		_, err = os.Stat(filepath.Join(firstPath(s), "node"))
	}
	require.NoError(t, err)
}
