package builtin

import (
	"context"
	"testing"

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
	autogold.Expect("").Equal(t, v)
}
