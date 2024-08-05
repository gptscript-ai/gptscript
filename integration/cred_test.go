package integration

import (
	"strings"
	"testing"
	"time"

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
	out, err := RunScript("scripts/cred_scopes.gpt", "--sub-tool", "oneOne")
	require.NoError(t, err)
	require.Contains(t, out, "good")

	out, err = RunScript("scripts/cred_scopes.gpt", "--sub-tool", "twoOne")
	require.NoError(t, err)
	require.Contains(t, out, "good")

	out, err = RunScript("scripts/cred_scopes.gpt", "--sub-tool", "twoTwo")
	require.NoError(t, err)
	require.Contains(t, out, "good")
}

// TestCredentialExpirationEnv tests a GPTScript with two credentials that expire at different times.
// One expires after two hours, and the other expires after one hour.
// This test makes sure that the GPTSCRIPT_CREDENTIAL_EXPIRATION environment variable is set to the nearer expiration time (1h).
func TestCredentialExpirationEnv(t *testing.T) {
	out, err := RunScript("scripts/cred_expiration.gpt")
	require.NoError(t, err)

	for _, line := range strings.Split(out, "\n") {
		if timestamp, found := strings.CutPrefix(line, "Expires: "); found {
			expiresTime, err := time.Parse(time.RFC3339, timestamp)
			require.NoError(t, err)
			require.True(t, time.Until(expiresTime) < time.Hour)
		}
	}
}
