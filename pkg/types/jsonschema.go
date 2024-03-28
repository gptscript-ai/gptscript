package types

import "encoding/json"

type AdditionalProperties struct {
	Has    *bool
	Schema *JSONSchema
}

func (a *AdditionalProperties) UnmarshalJSON(data []byte) error {
	if data[0] == '{' {
		a.Schema = &JSONSchema{}
		return json.Unmarshal(data, a.Schema)
	}
	return json.Unmarshal(data, a.Has)
}

func (a *AdditionalProperties) MarshalJSON() ([]byte, error) {
	if a.Schema != nil {
		return json.Marshal(a.Schema)
	}
	return json.Marshal(a.Has)
}

type JSONSchema struct {
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Ref         string      `json:"$ref,omitempty"`
	Items       *JSONSchema `json:"items,omitempty"`
	Enum        []string    `json:"enum,omitempty"`

	ID         string                `json:"$id,omitempty"`
	Title      string                `json:"title,omitempty"`
	Properties map[string]JSONSchema `json:"properties"`
	Required   []string              `json:"required,omitempty"`
	Defs       map[string]JSONSchema `json:"defs,omitempty"`

	AdditionalProperties AdditionalProperties `json:"additionalProperties,omitempty"`
}

func ObjectSchema(kv ...string) *JSONSchema {
	s := &JSONSchema{
		Type:       "object",
		Properties: map[string]JSONSchema{},
	}
	for i, v := range kv {
		if i%2 == 1 {
			s.Properties[kv[i-1]] = JSONSchema{
				Description: v,
				Type:        "string",
			}
		}
	}
	return s
}

type Type []string

func (t *Type) UnmarshalJSON(data []byte) error {
	switch data[0] {
	case '[':
		return json.Unmarshal(data, (*[]string)(t))
	case 'n':
		return json.Unmarshal(data, (*[]string)(t))
	default:
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = []string{s}
	}
	return nil
}

func (t *Type) MarshalJSON() ([]byte, error) {
	switch len(*t) {
	case 0:
		return json.Marshal(nil)
	case 1:
		return json.Marshal((*t)[0])
	default:
		return json.Marshal(*t)
	}
}
