package types

import (
	"encoding/json"
	"strings"
)

const (
	PromptURLEnvVar   = "GPTSCRIPT_PROMPT_URL"
	PromptTokenEnvVar = "GPTSCRIPT_PROMPT_TOKEN"
)

type Prompt struct {
	Message   string            `json:"message,omitempty"`
	Fields    Fields            `json:"fields,omitempty"`
	Sensitive bool              `json:"sensitive,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Field struct {
	Name        string   `json:"name,omitempty"`
	Sensitive   *bool    `json:"sensitive,omitempty"`
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type Fields []Field

// UnmarshalJSON will unmarshal the corresponding JSON object for Fields,
// or a comma-separated strings (for backwards compatibility).
func (f *Fields) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || f == nil {
		return nil
	}

	if b[0] == '[' {
		var arr []Field
		if err := json.Unmarshal(b, &arr); err != nil {
			return err
		}
		*f = arr
		return nil
	}

	var fields string
	if err := json.Unmarshal(b, &fields); err != nil {
		return err
	}

	if fields != "" {
		fieldsArr := strings.Split(fields, ",")
		*f = make([]Field, 0, len(fieldsArr))
		for _, field := range fieldsArr {
			*f = append(*f, Field{Name: strings.TrimSpace(field)})
		}
	}

	return nil
}

type field *Field

// UnmarshalJSON will unmarshal the corresponding JSON object for a Field,
// or a string (for backwards compatibility).
func (f *Field) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || f == nil {
		return nil
	}

	if b[0] == '{' {
		return json.Unmarshal(b, field(f))
	}

	return json.Unmarshal(b, &f.Name)
}
