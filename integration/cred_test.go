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

// TestStackedCredentialContexts tests creating, using, listing, showing, and deleting credentials when there are multiple contexts.
func TestStackedCredentialContexts(t *testing.T) {
	// First, test credential creation. We will create a credential called testcred in two different contexts called one and two.
	_, err := RunScript("scripts/cred_stacked.gpt", "--sub-tool", "testcred_one", "--credential-context", "one,two")
	require.NoError(t, err)

	_, err = RunScript("scripts/cred_stacked.gpt", "--sub-tool", "testcred_two", "--credential-context", "two")
	require.NoError(t, err)

	// Next, we try running the testcred_one tool. It should print the value of "testcred" in whichever context it finds the cred first.
	out, err := RunScript("scripts/cred_stacked.gpt", "--sub-tool", "testcred_one", "--credential-context", "one,two")
	require.NoError(t, err)
	require.Contains(t, out, "one")
	require.NotContains(t, out, "two")

	out, err = RunScript("scripts/cred_stacked.gpt", "--sub-tool", "testcred_one", "--credential-context", "two,one")
	require.NoError(t, err)
	require.Contains(t, out, "two")
	require.NotContains(t, out, "one")

	// Next, list credentials and specify both contexts. We should get the credential from the first specified context.
	out, err = GPTScriptExec("--credential-context", "one,two", "cred")
	require.NoError(t, err)
	require.Contains(t, out, "one")
	require.NotContains(t, out, "two")

	out, err = GPTScriptExec("--credential-context", "two,one", "cred")
	require.NoError(t, err)
	require.Contains(t, out, "two")
	require.NotContains(t, out, "one")

	// Next, try showing the credentials.
	out, err = GPTScriptExec("--credential-context", "one,two", "cred", "show", "testcred")
	require.NoError(t, err)
	require.Contains(t, out, "one")
	require.NotContains(t, out, "two")

	out, err = GPTScriptExec("--credential-context", "two,one", "cred", "show", "testcred")
	require.NoError(t, err)
	require.Contains(t, out, "two")
	require.NotContains(t, out, "one")

	// Make sure we get an error if we try to delete a credential with multiple contexts specified.
	_, err = GPTScriptExec("--credential-context", "one,two", "cred", "delete", "testcred")
	require.Error(t, err)

	// Now actually delete the credentials.
	_, err = GPTScriptExec("--credential-context", "one", "cred", "delete", "testcred")
	require.NoError(t, err)

	_, err = GPTScriptExec("--credential-context", "two", "cred", "delete", "testcred")
	require.NoError(t, err)
}
