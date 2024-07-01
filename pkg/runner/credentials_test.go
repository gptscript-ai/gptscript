package runner

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCredentialOverrides(t *testing.T) {
	cases := []struct {
		name      string
		envs      map[string]string
		in        []string
		out       map[string]map[string]string
		expectErr bool
	}{
		{
			name: "nil",
			in:   nil,
			out:  map[string]map[string]string{},
		},
		{
			name:      "empty",
			in:        []string{""},
			expectErr: true,
		},
		{
			name: "single cred, single env",
			envs: map[string]string{
				"ENV1": "VALUE1",
			},
			in: []string{"cred1:ENV1"},
			out: map[string]map[string]string{
				"cred1": {
					"ENV1": "VALUE1",
				},
			},
		},
		{
			name: "single cred, multiple envs",
			envs: map[string]string{
				"ENV1": "VALUE1",
				"ENV2": "VALUE2",
			},
			in: []string{"cred1:ENV1,ENV2"},
			out: map[string]map[string]string{
				"cred1": {
					"ENV1": "VALUE1",
					"ENV2": "VALUE2",
				},
			},
		},
		{
			name: "single cred, key value pairs",
			envs: map[string]string{
				"ENV1": "VALUE1",
				"ENV2": "VALUE2",
			},
			in: []string{"cred1:ENV1=OTHERVALUE1,ENV2=OTHERVALUE2"},
			out: map[string]map[string]string{
				"cred1": {
					"ENV1": "OTHERVALUE1",
					"ENV2": "OTHERVALUE2",
				},
			},
		},
		{
			name: "multiple creds, multiple envs",
			envs: map[string]string{
				"ENV1": "VALUE1",
				"ENV2": "VALUE2",
			},
			in: []string{"cred1:ENV1,ENV2", "cred2:ENV1,ENV2"},
			out: map[string]map[string]string{
				"cred1": {
					"ENV1": "VALUE1",
					"ENV2": "VALUE2",
				},
				"cred2": {
					"ENV1": "VALUE1",
					"ENV2": "VALUE2",
				},
			},
		},
		{
			name: "multiple creds, key value pairs",
			envs: map[string]string{
				"ENV1": "VALUE1",
				"ENV2": "VALUE2",
			},
			in: []string{"cred1:ENV1=OTHERVALUE1,ENV2=OTHERVALUE2", "cred2:ENV1=OTHERVALUE3,ENV2=OTHERVALUE4"},
			out: map[string]map[string]string{
				"cred1": {
					"ENV1": "OTHERVALUE1",
					"ENV2": "OTHERVALUE2",
				},
				"cred2": {
					"ENV1": "OTHERVALUE3",
					"ENV2": "OTHERVALUE4",
				},
			},
		},
		{
			name:      "invalid format",
			in:        []string{"cred1=ENV1,ENV2"},
			expectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			envs := tc.envs
			if envs == nil {
				envs = map[string]string{}
			}

			for k, v := range envs {
				_ = os.Setenv(k, v)
			}

			out, err := parseCredentialOverrides(tc.in)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, len(tc.out), len(out), "expected %d creds, but got %d", len(tc.out), len(out))
			require.Equal(t, tc.out, out, "expected output %v, but got %v", tc.out, out)
		})
	}
}
