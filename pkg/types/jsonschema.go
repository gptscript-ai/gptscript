package types

import humav2 "github.com/danielgtaylor/huma/v2"

func ObjectSchema(kv ...string) *humav2.Schema {
	s := &humav2.Schema{
		Type:       humav2.TypeObject,
		Properties: make(map[string]*humav2.Schema, len(kv)/2),
	}
	for i, v := range kv {
		if i%2 == 1 {
			s.Properties[kv[i-1]] = &humav2.Schema{
				Description: v,
				Type:        humav2.TypeString,
			}
		}
	}
	return s
}
