package credentials

import (
	"context"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDBStore(t *testing.T) {
	const credCtx = "default"
	credential := Credential{
		Context:      credCtx,
		ToolName:     "mytestcred",
		Type:         CredentialTypeTool,
		Env:          map[string]string{"ASDF": "yeet"},
		RefreshToken: "myrefreshtoken",
	}

	cfg, _ := config.ReadCLIConfig("")

	store, err := NewDBStore(context.Background(), cfg, []string{credCtx})
	require.NoError(t, err)

	require.NoError(t, store.Add(context.Background(), credential))

	cred, found, err := store.Get(context.Background(), credential.ToolName)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, credential.Env, cred.Env)
	require.Equal(t, credential.RefreshToken, cred.RefreshToken)

	list, err := store.List(context.Background())
	require.NoError(t, err)
	require.Greater(t, len(list), 0)
	for _, c := range list {
		if c.Context == credCtx && c.ToolName == credential.ToolName {
			require.Equal(t, credential.Env, c.Env)
			require.Equal(t, credential.RefreshToken, c.RefreshToken)
		}
	}

	require.NoError(t, store.Remove(context.Background(), credential.ToolName))
}
