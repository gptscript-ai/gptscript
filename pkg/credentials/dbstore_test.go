package credentials

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDBStore(t *testing.T) {
	const credCtx = "testing"

	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	require.NoError(t, err)

	credential := Credential{
		Context:      credCtx,
		ToolName:     fmt.Sprintf("%x", bytes),
		Type:         CredentialTypeTool,
		Env:          map[string]string{"ENV_VAR": "value"},
		RefreshToken: "myrefreshtoken",
	}

	cfg, _ := config.ReadCLIConfig("")

	// Set up the store
	store, err := NewDBStore(context.Background(), cfg, []string{credCtx})
	require.NoError(t, err)

	// Create the credential
	require.NoError(t, store.Add(context.Background(), credential))

	// Get the credential
	cred, found, err := store.Get(context.Background(), credential.ToolName)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, credential.Env, cred.Env)
	require.Equal(t, credential.RefreshToken, cred.RefreshToken)

	// List credentials and check for it
	list, err := store.List(context.Background())
	require.NoError(t, err)
	require.Greater(t, len(list), 0)

	found = false
	for _, c := range list {
		if c.Context == credCtx && c.ToolName == credential.ToolName {
			require.Equal(t, credential.Env, c.Env)
			require.Equal(t, credential.RefreshToken, c.RefreshToken)
			found = true
			break
		}
	}
	require.True(t, found)

	// Delete the credential
	require.NoError(t, store.Remove(context.Background(), credential.ToolName))
}
