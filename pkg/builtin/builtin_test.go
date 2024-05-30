package builtin

import (
	"context"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
)

func TestSysGetenv(t *testing.T) {
	v, err := SysGetenv(context.Background(), []string{
		"MAGIC=VALUE",
	}, `{"name":"MAGIC"}`)
	require.NoError(t, err)
	autogold.Expect("VALUE").Equal(t, v)

	v, err = SysGetenv(context.Background(), []string{
		"MAGIC=VALUE",
	}, `{"name":"MAGIC2"}`)
	require.NoError(t, err)
	autogold.Expect("MAGIC2 is not set or has no value").Equal(t, v)
}

func TestDisplayCoverage(t *testing.T) {
	for _, tool := range ListTools() {
		_, err := types.ToSysDisplayString(tool.ID, nil)
		require.NoError(t, err)
	}
}
