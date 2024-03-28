package engine

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathParameterSerialization(t *testing.T) {
	input := struct {
		Value  int               `json:"v"`
		Array  []string          `json:"a"`
		Object map[string]string `json:"o"`
	}{
		Value:  42,
		Array:  []string{"foo", "bar", "baz"},
		Object: map[string]string{"qux": "quux", "corge": "grault"},
	}
	inputStr, err := json.Marshal(input)
	require.NoError(t, err)

	path := "/mypath/{v}/{a}/{o}"

	tests := []struct {
		name          string
		style         string
		explode       bool
		expectedPaths []string // We use multiple expected paths due to randomness in map iteration
	}{
		{
			name:    "simple + no explode",
			style:   "simple",
			explode: false,
			expectedPaths: []string{
				"/mypath/42/foo,bar,baz/qux,quux,corge,grault",
				"/mypath/42/foo,bar,baz/corge,grault,qux,quux",
			},
		},
		{
			name:    "simple + explode",
			style:   "simple",
			explode: true,
			expectedPaths: []string{
				"/mypath/42/foo,bar,baz/qux=quux,corge=grault",
				"/mypath/42/foo,bar,baz/corge=grault,qux=quux",
			},
		},
		{
			name:    "label + no explode",
			style:   "label",
			explode: false,
			expectedPaths: []string{
				"/mypath/.42/.foo,bar,baz/.qux,quux,corge,grault",
				"/mypath/.42/.foo,bar,baz/.corge,grault,qux,quux",
			},
		},
		{
			name:    "label + explode",
			style:   "label",
			explode: true,
			expectedPaths: []string{
				"/mypath/.42/.foo.bar.baz/.qux=quux.corge=grault",
				"/mypath/.42/.foo.bar.baz/.corge=grault.qux=quux",
			},
		},
		{
			name:    "matrix + no explode",
			style:   "matrix",
			explode: false,
			expectedPaths: []string{
				"/mypath/;v=42/;a=foo,bar,baz/;o=qux,quux,corge,grault",
				"/mypath/;v=42/;a=foo,bar,baz/;o=corge,grault,qux,quux",
			},
		},
		{
			name:    "matrix + explode",
			style:   "matrix",
			explode: true,
			expectedPaths: []string{
				"/mypath/;v=42/;a=foo;a=bar;a=baz/;qux=quux;corge=grault",
				"/mypath/;v=42/;a=foo;a=bar;a=baz/;corge=grault;qux=quux",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := path
			params := getParameters(test.style, test.explode)
			path = handlePathParameters(path, params, string(inputStr))
			require.Contains(t, test.expectedPaths, path)
		})
	}
}

func TestQueryParameterSerialization(t *testing.T) {
	input := struct {
		Value  int               `json:"v"`
		Array  []string          `json:"a"`
		Object map[string]string `json:"o"`
	}{
		Value:  42,
		Array:  []string{"foo", "bar", "baz"},
		Object: map[string]string{"qux": "quux", "corge": "grault"},
	}
	inputStr, err := json.Marshal(input)
	require.NoError(t, err)

	tests := []struct {
		name            string
		input           string
		param           Parameter
		expectedQueries []string // We use multiple expected queries due to randomness in map iteration
	}{
		{
			name:  "value",
			input: string(inputStr),
			param: Parameter{
				Name: "v",
			},
			expectedQueries: []string{"v=42"},
		},
		{
			name:  "array form + explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "form",
				Explode: boolPointer(true),
			},
			expectedQueries: []string{"a=foo&a=bar&a=baz"},
		},
		{
			name:  "array form + no explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "form",
				Explode: boolPointer(false),
			},
			expectedQueries: []string{"a=foo%2Cbar%2Cbaz"}, // %2C is a comma
		},
		{
			name:  "array spaceDelimited + explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "spaceDelimited",
				Explode: boolPointer(true),
			},
			expectedQueries: []string{"a=foo&a=bar&a=baz"},
		},
		{
			name:  "array spaceDelimited + no explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "spaceDelimited",
				Explode: boolPointer(false),
			},
			expectedQueries: []string{"a=foo+bar+baz"},
		},
		{
			name:  "array pipeDelimited + explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "pipeDelimited",
				Explode: boolPointer(true),
			},
			expectedQueries: []string{"a=foo&a=bar&a=baz"},
		},
		{
			name:  "array pipeDelimited + no explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "a",
				Style:   "pipeDelimited",
				Explode: boolPointer(false),
			},
			expectedQueries: []string{"a=foo%7Cbar%7Cbaz"}, // %7C is a pipe
		},
		{
			name:  "object form + explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "o",
				Style:   "form",
				Explode: boolPointer(true),
			},
			expectedQueries: []string{
				"qux=quux&corge=grault",
				"corge=grault&qux=quux",
			},
		},
		{
			name:  "object form + no explode",
			input: string(inputStr),
			param: Parameter{
				Name:    "o",
				Style:   "form",
				Explode: boolPointer(false),
			},
			expectedQueries: []string{ // %2C is a comma
				"o=qux%2Cquux%2Ccorge%2Cgrault",
				"o=corge%2Cgrault%2Cqux%2Cquux",
			},
		},
		{
			name:  "object deepObject",
			input: string(inputStr),
			param: Parameter{
				Name:  "o",
				Style: "deepObject",
			},
			expectedQueries: []string{ // %5B is a [ and %5D is a ]
				"o%5Bqux%5D=quux&o%5Bcorge%5D=grault",
				"o%5Bcorge%5D=grault&o%5Bqux%5D=quux",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q := handleQueryParameters(url.Values{}, []Parameter{test.param}, test.input)
			require.Contains(t, test.expectedQueries, q.Encode())
		})
	}
}

func getParameters(style string, explode bool) []Parameter {
	return []Parameter{
		{
			Name:    "v",
			Style:   style,
			Explode: &explode,
		},
		{
			Name:    "a",
			Style:   style,
			Explode: &explode,
		},
		{
			Name:    "o",
			Style:   style,
			Explode: &explode,
		},
	}
}

func boolPointer(b bool) *bool {
	return &b
}
