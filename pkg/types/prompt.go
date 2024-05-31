package types

const PromptURLEnvVar = "GPTSCRIPT_PROMPT_URL"

type Prompt struct {
	Message   string   `json:"message,omitempty"`
	Fields    []string `json:"fields,omitempty"`
	Sensitive bool     `json:"sensitive,omitempty"`
}
