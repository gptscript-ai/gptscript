package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONSchemaAdditionalProperties(t *testing.T) {
	boolVersion := &JSONSchema{
		AdditionalProperties: AdditionalProperties{
			Has: boolPointer(true),
		},
		Properties: map[string]JSONSchema{},
	}
	raw, err := json.Marshal(boolVersion)
	require.NoError(t, err)
	require.JSONEq(t, `{"additionalProperties":true, "properties":{}}`, string(raw))

	schemaVersion := &JSONSchema{
		AdditionalProperties: AdditionalProperties{
			Schema: &JSONSchema{
				Type:       "string",
				Properties: map[string]JSONSchema{},
			},
		},
		Properties: map[string]JSONSchema{},
	}
	raw, err = json.Marshal(schemaVersion)
	require.NoError(t, err)
	require.JSONEq(t, `{"additionalProperties":{"type":"string", "properties":{}, "additionalProperties":null}, "properties":{}}`, string(raw))
}

func boolPointer(b bool) *bool {
	return &b
}
