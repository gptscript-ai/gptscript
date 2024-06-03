package busybox

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS != "windows" {
		t.Skip()
	}

	r := Runtime{}

	s, err := r.Setup(context.Background(), testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(firstPath(s), "busybox.exe"))
	if errors.Is(err, fs.ErrNotExist) {
		_, err = os.Stat(filepath.Join(firstPath(s), "busybox"))
	}
	require.NoError(t, err)
}
