// File: cmd_test.go
package engine

import "testing"

func TestSplitByQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "NoQuotes",
			input:    "Hello World",
			expected: []string{"Hello World"},
		},
		{
			name:     "ValidQuote",
			input:    `"Hello" "World"`,
			expected: []string{``, `"Hello"`, ` `, `"World"`},
		},
		{
			name:     "ValidQuoteWithEscape",
			input:    `"Hello\" World"`,
			expected: []string{``, `"Hello\" World"`},
		},
		{
			name:     "Nothing",
			input:    "",
			expected: []string{},
		},
		{
			name:     "SpaceInsideQuote",
			input:    `"Hello World"`,
			expected: []string{``, `"Hello World"`},
		},
		{
			name:     "SingleChar",
			input:    "H",
			expected: []string{"H"},
		},
		{
			name:     "SingleQuote",
			input:    `"Hello`,
			expected: []string{``, ``, `"Hello`},
		},
		{
			name:     "ThreeQuotes",
			input:    `Test "Hello "World" End\"`,
			expected: []string{`Test `, `"Hello "`, `World`, ``, `" End\"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitByQuotes(tt.input)
			if !equal(got, tt.expected) {
				t.Errorf("splitByQuotes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// Helper function to assert equality of two string slices.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Testing for replaceVariablesForInterpreter
func TestReplaceVariablesForInterpreter(t *testing.T) {
	tests := []struct {
		name        string
		interpreter string
		envMap      map[string]string
		expected    []string
		shouldFail  bool
	}{
		{
			name:        "No quotes",
			interpreter: "/bin/bash -c ${COMMAND} tail",
			envMap:      map[string]string{"COMMAND": "echo Hello!"},
			expected:    []string{"/bin/bash", "-c", "echo", "Hello!", "tail"},
		},
		{
			name:        "Quotes Variables",
			interpreter: `/bin/bash -c "${COMMAND}" tail`,
			envMap:      map[string]string{"COMMAND": "Hello, World!"},
			expected:    []string{"/bin/bash", "-c", "Hello, World!", "tail"},
		},
		{
			name:        "Double escape",
			interpreter: `/bin/bash -c "${COMMAND}" ${TWO} tail`,
			envMap: map[string]string{
				"COMMAND": "Hello, World!",
				"TWO":     "${COMMAND}",
			},
			expected: []string{"/bin/bash", "-c", "Hello, World!", "${COMMAND}", "tail"},
		},
		{
			name:        "aws cli issue",
			interpreter: "aws ${ARGS}",
			envMap: map[string]string{
				"ARGS": `ec2 describe-instances --region us-east-1 --query 'Reservations[*].Instances[*].{Instance:InstanceId,State:State.Name}'`,
			},
			expected: []string{
				`aws`,
				`ec2`,
				`describe-instances`,
				`--region`, `us-east-1`,
				`--query`, `Reservations[*].Instances[*].{Instance:InstanceId,State:State.Name}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceVariablesForInterpreter(tt.interpreter, tt.envMap)
			if (err != nil) != tt.shouldFail {
				t.Errorf("replaceVariablesForInterpreter() error = %v, want %v", err, tt.shouldFail)
				return
			}
			if !equal(got, tt.expected) {
				t.Errorf("replaceVariablesForInterpreter() = %v, want %v", got, tt.expected)
			}
		})
	}
}
