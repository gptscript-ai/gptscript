package python

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
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
		Version: "3.12",
	}

	s, err := r.Setup(context.Background(), testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(firstPath(s), "python.exe"))
	if errors.Is(err, os.ErrNotExist) {
		_, err = os.Stat(filepath.Join(firstPath(s), "python"))
	}
	require.NoError(t, err)
}
