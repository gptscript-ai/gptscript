package types

const (
	PromptURLEnvVar   = "GPTSCRIPT_PROMPT_URL"
	PromptTokenEnvVar = "GPTSCRIPT_PROMPT_TOKEN"
)

type Prompt struct {
	Message   string            `json:"message,omitempty"`
	Fields    []string          `json:"fields,omitempty"`
	Sensitive bool              `json:"sensitive,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
