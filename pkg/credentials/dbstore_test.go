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

	cfg, err := config.ReadCLIConfig("")
	require.NoError(t, err)

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

func TestDBStoreStackedContexts(t *testing.T) {
	const (
		credCtx1 = "testing1"
		credCtx2 = "testing2"
	)

	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	require.NoError(t, err)

	credential1 := Credential{
		Context:  credCtx1,
		ToolName: fmt.Sprintf("%x", bytes),
		Type:     CredentialTypeTool,
		Env:      map[string]string{"ENV_VAR": "value"},
	}

	credential2 := Credential{
		Context:  credCtx2,
		ToolName: fmt.Sprintf("%x", bytes),
		Type:     CredentialTypeTool,
		Env:      map[string]string{"ENV_VAR": "value"},
	}

	cfg, err := config.ReadCLIConfig("")
	require.NoError(t, err)

	// Set up the stores
	store1, err := NewDBStore(context.Background(), cfg, []string{credCtx1})
	require.NoError(t, err)
	store2, err := NewDBStore(context.Background(), cfg, []string{credCtx2})
	require.NoError(t, err)

	// Create both credentials
	require.NoError(t, store1.Add(context.Background(), credential1))
	require.NoError(t, store2.Add(context.Background(), credential2))

	// Set up a store with both contexts
	storeBoth, err := NewDBStore(context.Background(), cfg, []string{credCtx1, credCtx2})
	require.NoError(t, err)

	// Get the credential. We should get credential1.
	cred, found, err := storeBoth.Get(context.Background(), credential1.ToolName)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, credential1.ToolName, cred.ToolName)
	require.Equal(t, credential1.Context, cred.Context)
	require.Equal(t, credential1.Env, cred.Env)

	// List credentials. We should only get credential1.
	list, err := storeBoth.List(context.Background())
	require.NoError(t, err)

	found = false
	for _, c := range list {
		if c.ToolName == credential1.ToolName {
			require.Equal(t, credential1.Env, c.Env)
			require.Equal(t, credential1.Context, c.Context)
			found = true
			break
		} else {
			require.Fail(t, "unexpected credential found")
		}
	}
	require.True(t, found)

	// Delete both credentials
	require.NoError(t, store1.Remove(context.Background(), credential1.ToolName))
	require.NoError(t, store2.Remove(context.Background(), credential2.ToolName))
}
