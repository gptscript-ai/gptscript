package mcp

import (
	"encoding/json"
)

type ClientCapabilities struct {
	Roots       *RootsCapability `json:"roots,omitempty"`
	Sampling    *struct{}        `json:"sampling,omitzero"`
	Elicitation *struct{}        `json:"elicitation,omitzero"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Experimental map[string]any             `json:"experimental,omitempty"`
	Logging      *struct{}                  `json:"logging,omitempty"`
	Prompts      *PromptsServerCapability   `json:"prompts,omitempty"`
	Resources    *ResourcesServerCapability `json:"resources,omitempty"`
	Tools        *ToolsServerCapability     `json:"tools,omitempty"`
}

type ToolsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

type PromptsServerCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ResourcesServerCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Instructions    string             `json:"instructions"`
}

type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type PingRequest struct {
}

type PingResult struct {
}

type ElicitResult struct {
	// Action must be one of "accept", "decline", "cancel"
	Action  string         `json:"action"`
	Content map[string]any `json:"content,omitempty"`
}

type ElicitRequest struct {
	Message         string          `json:"message,omitempty"`
	RequestedSchema PrimitiveSchema `json:"requestedSchema,omitzero"`
	Meta            json.RawMessage `json:"_meta,omitzero"`
}

type PrimitiveSchema struct {
	// Type must be "object" only
	Type       string                       `json:"type"`
	Properties map[string]PrimitiveProperty `json:"properties"`
}

type PrimitiveProperty struct {
	// Type must be one of "string", "number", "boolean", "enum", "integer"
	Type        string       `json:"type"`
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	MinLength   *int         `json:"minLength,omitempty"`
	MaxLength   *int         `json:"maxLength,omitempty"`
	Minimum     *json.Number `json:"minimum,omitempty"`
	Maximum     *json.Number `json:"maximum,omitempty"`
	Default     *bool        `json:"default,omitempty"`
	Enum        []string     `json:"enum,omitempty"`
	EnumNames   []string     `json:"enumNames,omitempty"`
	// Format must be one of "date-time", "email", "uri", "date"
	Format string `json:"format,omitempty"`
}

type ModelPreferences struct {
	Hints                []ModelHint `json:"hints,omitzero"`
	CostPriority         *float64    `json:"costPriority"`
	SpeedPriority        *float64    `json:"speedPriority"`
	IntelligencePriority *float64    `json:"intelligencePriority"`
}

type ModelHint struct {
	Name string `json:"name"`
}
type CreateMessageRequest struct {
	Messages         []SamplingMessage `json:"messages,omitzero"`
	ModelPreferences ModelPreferences  `json:"modelPreferences,omitzero"`
	SystemPrompt     string            `json:"systemPrompt,omitzero"`
	IncludeContext   string            `json:"includeContext,omitempty"`
	MaxTokens        int               `json:"maxTokens,omitempty"`
	Temperature      *json.Number      `json:"temperature,omitempty"`
	StopSequences    []string          `json:"stopSequences,omitzero"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
}

type ListRootsRequest struct {
}

type ListRootsResult struct {
	Roots []Root `json:"roots"`
}

type Root struct {
	URI  string `json:"uri"`
	Name string `json:"name,omitempty"`
}

type LoggingMessage struct {
	Level  string `json:"level"`
	Logger string `json:"logger,omitempty"`
	Data   any    `json:"data"`
}

type SamplingMessage struct {
	Role    string  `json:"role,omitempty"`
	Content Content `json:"content,omitempty"`
}

type Content struct {
	Type string `json:"type,omitempty"`

	// Text is set when type is "text"
	Text string `json:"text,omitempty"`

	// StructuredContent is set when the content is structured. The spec isn't clear when, but it's
	// likely to only be set when type is "text".
	StructuredContent any `json:"structuredContent,omitempty"`

	// Data is set when type is "image" or "audio"
	Data string `json:"data,omitempty"`
	// MIMEType is set when type is "image" or "audio"
	MIMEType string `json:"mimeType,omitempty"`

	// Resource is set when type is "resource"
	Resource *EmbeddedResource `json:"resource,omitempty"`
}

func (c Content) MarshalJSON() ([]byte, error) {
	type Alias Content
	if c.Type == "" {
		if c.Resource != nil {
			c.Type = "resource"
		} else if c.Text != "" || c.StructuredContent != nil {
			c.Type = "text"
		} else if c.Data != "" {
			c.Type = "image"
		}
	}
	return json.Marshal((*Alias)(&c))
}

type CreateMessageResult struct {
	Content    Content `json:"content,omitempty"`
	Role       string  `json:"role,omitempty"`
	Model      string  `json:"model,omitempty"`
	StopReason string  `json:"stopReason,omitempty"`
}

func (c *Content) ToImageURL() string {
	return "data:" + c.MIMEType + ";base64," + c.Data
}

type EmbeddedResource struct {
	URI      string `json:"uri,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type Tool struct {
	Name         string           `json:"name"`
	Description  string           `json:"description,omitempty"`
	InputSchema  json.RawMessage  `json:"inputSchema,omitzero"`
	OutputSchema json.RawMessage  `json:"outputSchema,omitzero"`
	Annotations  *ToolAnnotations `json:"annotations,omitempty"`
}

type ToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
}

func (t ToolAnnotations) IsOpenWorld() bool {
	if t.OpenWorldHint == nil {
		return true
	}
	return *t.OpenWorldHint
}

func (t ToolAnnotations) IsDestructive() bool {
	if t.DestructiveHint == nil {
		return true
	}
	return *t.DestructiveHint
}

type CallToolResult struct {
	IsError bool      `json:"isError"`
	Content []Content `json:"content,omitzero"`
}

type CallToolRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

var EmptyObjectSchema = json.RawMessage(`{"type": "object", "properties": {}, "additionalProperties": false, "required": []}`)

type ListToolsRequest struct {
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type GetPromptRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

type ReadResourceRequest struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type ListResourceTemplatesRequest struct {
}

type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

type ResourceTemplate struct {
	URITemplate string       `json:"uriTemplate"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	MimeType    string       `json:"mimeType,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

type ListResourcesRequest struct {
}

type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

type Resource struct {
	URI         string       `json:"uri"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	MimeType    string       `json:"mimeType,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Size        int64        `json:"size,omitempty"`
}

type Annotations struct {
	Audience []string    `json:"audience,omitempty"`
	Priority json.Number `json:"priority,omitempty"`
}

type ListPromptsRequest struct {
}

type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Notification struct {
}

type NotificationProgressRequest struct {
	ProgressToken any            `json:"progressToken"`
	Progress      json.Number    `json:"progress"`
	Total         *json.Number   `json:"total,omitempty"`
	Message       string         `json:"message,omitempty"`
	Meta          map[string]any `json:"_meta,omitzero"`
}
