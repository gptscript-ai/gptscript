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
