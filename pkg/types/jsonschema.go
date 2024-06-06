package types

import (
	"github.com/getkin/kin-openapi/openapi3"
)

func ObjectSchema(kv ...string) *openapi3.Schema {
	s := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: openapi3.Schemas{},
	}
	for i, v := range kv {
		if i%2 == 1 {
			s.Properties[kv[i-1]] = &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Description: v,
					Type:        &openapi3.Types{"string"},
				},
			}
		}
	}
	return s
}
