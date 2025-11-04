//nolint:revive
package types

import "github.com/modelcontextprotocol/go-sdk/jsonschema"

func ObjectSchema(kv ...string) *jsonschema.Schema {
	s := &jsonschema.Schema{
		Type:       "object",
		Properties: make(map[string]*jsonschema.Schema, len(kv)/2),
	}
	for i, v := range kv {
		if i%2 == 1 {
			s.Properties[kv[i-1]] = &jsonschema.Schema{
				Description: v,
				Type:        "string",
			}
		}
	}
	return s
}
