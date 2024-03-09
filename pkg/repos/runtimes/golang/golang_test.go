package golang

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCacheHome = lo.Must(xdg.CacheFile("gptscript-test-cache/runtime"))
)

func TestRuntime(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll("testdata/bin")
	})
	r := Runtime{
		Version: "1.22.1",
	}

	s, err := r.Setup(context.Background(), testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	p, v, _ := strings.Cut(s[0], "=")
	v, _, _ = strings.Cut(v, string(filepath.ListSeparator))
	assert.Equal(t, "PATH", p)
	_, err = os.Stat(filepath.Join(v, "gofmt"))
	assert.NoError(t, err)
}
