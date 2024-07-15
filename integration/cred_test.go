package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGPTScriptCredential(t *testing.T) {
	out, err := GPTScriptExec("cred")
	require.NoError(t, err)
	require.Contains(t, out, "CREDENTIAL")
}

// TestCredentialScopes makes sure that environment variables set by credential tools and shared credential tools
// are only available to the correct tools. See scripts/credscopes.gpt for more details.
func TestCredentialScopes(t *testing.T) {
	out, err := RunScript("scripts/credscopes.gpt", "--sub-tool", "oneOne")
	require.NoError(t, err)
	require.Contains(t, out, "good")

	out, err = RunScript("scripts/credscopes.gpt", "--sub-tool", "twoOne")
	require.NoError(t, err)
	require.Contains(t, out, "good")

	out, err = RunScript("scripts/credscopes.gpt", "--sub-tool", "twoTwo")
	require.NoError(t, err)
	require.Contains(t, out, "good")
}
