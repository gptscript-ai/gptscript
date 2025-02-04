package types

import (
	"reflect"
	"testing"
)

func TestFieldUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		expected  Field
		expectErr bool
	}{
		{
			name:      "valid single Field object JSON",
			input:     []byte(`{"name":"field1","sensitive":true,"description":"A test field"}`),
			expected:  Field{Name: "field1", Sensitive: boolPtr(true), Description: "A test field"},
			expectErr: false,
		},
		{
			name:      "valid Field name as string",
			input:     []byte(`"field1"`),
			expected:  Field{Name: "field1"},
			expectErr: false,
		},
		{
			name:      "empty input",
			input:     []byte(``),
			expected:  Field{},
			expectErr: false,
		},
		{
			name:      "invalid JSON object",
			input:     []byte(`{"name":"field1","sensitive":"not_boolean"}`),
			expected:  Field{Name: "field1", Sensitive: new(bool)},
			expectErr: true,
		},
		{
			name:      "extra unknown fields in JSON object",
			input:     []byte(`{"name":"field1","unknown":"field","sensitive":false}`),
			expected:  Field{Name: "field1", Sensitive: boolPtr(false)},
			expectErr: false,
		},
		{
			name:      "malformed JSON",
			input:     []byte(`{"name":"field1","sensitive":true`),
			expected:  Field{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var field Field
			err := field.UnmarshalJSON(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("UnmarshalJSON() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !reflect.DeepEqual(field, tt.expected) {
				t.Errorf("UnmarshalJSON() = %v, expected %v", field, tt.expected)
			}
		})
	}
}

func TestFieldsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		expected  Fields
		expectErr bool
	}{
		{
			name:      "empty input",
			input:     nil,
			expected:  nil,
			expectErr: false,
		},
		{
			name:      "nil pointer",
			input:     nil,
			expected:  nil,
			expectErr: false,
		},
		{
			name:      "valid JSON array",
			input:     []byte(`[{"Name":"field1"},{"Name":"field2"}]`),
			expected:  Fields{{Name: "field1"}, {Name: "field2"}},
			expectErr: false,
		},
		{
			name:      "single string input",
			input:     []byte(`"field1,field2,field3"`),
			expected:  Fields{{Name: "field1"}, {Name: "field2"}, {Name: "field3"}},
			expectErr: false,
		},
		{
			name:      "trim spaces in single string input",
			input:     []byte(`"field1, field2 ,  field3  "`),
			expected:  Fields{{Name: "field1"}, {Name: "field2"}, {Name: "field3"}},
			expectErr: false,
		},
		{
			name:      "invalid JSON array",
			input:     []byte(`[{"Name":"field1"},{"Name":field2}]`),
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "invalid single string",
			input:     []byte(`1234`),
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "empty array",
			input:     []byte(`[]`),
			expected:  Fields{},
			expectErr: false,
		},
		{
			name:      "empty string",
			input:     []byte(`""`),
			expected:  nil,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fields Fields
			err := fields.UnmarshalJSON(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("UnmarshalJSON() error = %v, expectErr %v", err, tt.expectErr)
			}
			if !reflect.DeepEqual(fields, tt.expected) {
				t.Errorf("UnmarshalJSON() = %v, expected %v", fields, tt.expected)
			}
		})
	}
}
