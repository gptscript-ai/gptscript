package python

import (
	"context"
	"os"
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
	r := Runtime{
		Version: "3.12",
	}

	s, err := r.Setup(context.Background(), testCacheHome, "testdata", os.Environ())
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(s[0], "/bin"), "missing /bin: %s", s)
}
