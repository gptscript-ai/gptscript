package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCredentialArgs(t *testing.T) {
	tests := []struct {
		name          string
		toolName      string
		input         string
		expectedName  string
		expectedAlias string
		expectedArgs  map[string]string
		wantErr       bool
	}{
		{
			name:          "empty",
			toolName:      "",
			expectedName:  "",
			expectedAlias: "",
		},
		{
			name:          "tool name only",
			toolName:      "myCredentialTool",
			expectedName:  "myCredentialTool",
			expectedAlias: "",
		},
		{
			name:          "tool name and alias",
			toolName:      "myCredentialTool as myAlias",
			expectedName:  "myCredentialTool",
			expectedAlias: "myAlias",
		},
		{
			name:          "tool name with one arg",
			toolName:      "myCredentialTool with value1 as arg1",
			expectedName:  "myCredentialTool",
			expectedAlias: "",
			expectedArgs: map[string]string{
				"arg1": "value1",
			},
		},
		{
			name:          "tool name with two args",
			toolName:      "myCredentialTool with value1 as arg1 and value2 as arg2",
			expectedName:  "myCredentialTool",
			expectedAlias: "",
			expectedArgs: map[string]string{
				"arg1": "value1",
				"arg2": "value2",
			},
		},
		{
			name:          "tool name with alias and one arg",
			toolName:      "myCredentialTool as myAlias with value1 as arg1",
			expectedName:  "myCredentialTool",
			expectedAlias: "myAlias",
			expectedArgs: map[string]string{
				"arg1": "value1",
			},
		},
		{
			name:          "tool name with alias and two args",
			toolName:      "myCredentialTool as myAlias with value1 as arg1 and value2 as arg2",
			expectedName:  "myCredentialTool",
			expectedAlias: "myAlias",
			expectedArgs: map[string]string{
				"arg1": "value1",
				"arg2": "value2",
			},
		},
		{
			name:          "tool name with quoted args",
			toolName:      `myCredentialTool with "value one" as arg1 and "value two" as arg2`,
			expectedName:  "myCredentialTool",
			expectedAlias: "",
			expectedArgs: map[string]string{
				"arg1": "value one",
				"arg2": "value two",
			},
		},
		{
			name:          "tool name with arg references",
			toolName:      `myCredentialTool with ${var1} as arg1 and ${var2} as arg2`,
			input:         `{"var1": "value1", "var2": "value2"}`,
			expectedName:  "myCredentialTool",
			expectedAlias: "",
			expectedArgs: map[string]string{
				"arg1": "value1",
				"arg2": "value2",
			},
		},
		{
			name:     "tool name with alias but no 'as' (invalid)",
			toolName: "myCredentialTool myAlias",
			wantErr:  true,
		},
		{
			name:     "tool name with 'as' but no alias (invalid)",
			toolName: "myCredentialTool as",
			wantErr:  true,
		},
		{
			name:     "tool with 'with' but no args (invalid)",
			toolName: "myCredentialTool with",
			wantErr:  true,
		},
		{
			name:     "tool with args but no 'with' (invalid)",
			toolName: "myCredentialTool value1 as arg1",
			wantErr:  true,
		},
		{
			name:     "tool with trailing 'and' (invalid)",
			toolName: "myCredentialTool with value1 as arg1 and",
			wantErr:  true,
		},
		{
			name:     "tool with quoted arg but the quote is unterminated (invalid)",
			toolName: `myCredentialTool with "value one" as arg1 and "value two as arg2`,
			wantErr:  true,
		},
		{
			name:          "invalid input",
			toolName:      "myCredentialTool",
			input:         `{"asdf":"asdf"`,
			expectedName:  "myCredentialTool",
			expectedAlias: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalName, alias, args, err := ParseCredentialArgs(tt.toolName, tt.input)
			if tt.wantErr {
				require.Error(t, err, "expected an error but got none")
				return
			}

			require.NoError(t, err, "did not expect an error but got one")
			require.Equal(t, tt.expectedName, originalName, "unexpected original name")
			require.Equal(t, tt.expectedAlias, alias, "unexpected alias")
			require.Equal(t, len(tt.expectedArgs), len(args), "unexpected number of args")

			for k, v := range tt.expectedArgs {
				assert.Equal(t, v, args[k], "unexpected value for args[%s]", k)
			}
		})
	}
}
