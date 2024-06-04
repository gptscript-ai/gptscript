package types

import "testing"

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
			name:     "invalid input",
			toolName: "myCredentialTool",
			input:    `{"asdf":"asdf"`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalName, alias, args, err := ParseCredentialArgs(tt.toolName, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCredentialArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if originalName != tt.expectedName {
				t.Errorf("ParseCredentialArgs() originalName = %v, expectedName %v", originalName, tt.expectedName)
			}
			if alias != tt.expectedAlias {
				t.Errorf("ParseCredentialArgs() alias = %v, expectedAlias %v", alias, tt.expectedAlias)
			}
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("ParseCredentialArgs() args = %v, expectedArgs %v", args, tt.expectedArgs)
			}
			for k, v := range tt.expectedArgs {
				if args[k] != v {
					t.Errorf("ParseCredentialArgs() args[%s] = %v, expectedArgs[%s] %v", k, args[k], k, v)
				}
			}
		})
	}
}
