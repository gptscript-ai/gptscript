package credentials

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/config/types"
)

const ctxSeparator = "///"

type CredentialType string

const (
	CredentialTypeTool          CredentialType = "tool"
	CredentialTypeModelProvider CredentialType = "modelProvider"
)

type Credential struct {
	Context  string            `json:"context"`
	ToolName string            `json:"toolName"`
	Type     CredentialType    `json:"type"`
	Env      map[string]string `json:"env"`
}

func (c Credential) toDockerAuthConfig() (types.AuthConfig, error) {
	env, err := json.Marshal(c.Env)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      string(c.Type),
		Password:      string(env),
		ServerAddress: toolNameWithCtx(c.ToolName, c.Context),
	}, nil
}

func credentialFromDockerAuthConfig(authCfg types.AuthConfig) (Credential, error) {
	var env map[string]string
	if err := json.Unmarshal([]byte(authCfg.Password), &env); err != nil {
		return Credential{}, err
	}

	// We used to hardcode the username as "gptscript" before CredentialType was introduced, so
	// check for that here.
	credType := authCfg.Username
	if credType == "gptscript" {
		credType = string(CredentialTypeTool)
	}

	// If it's a tool credential or sys.openai, remove the http[s] prefix.
	address := authCfg.ServerAddress
	if credType == string(CredentialTypeTool) || strings.HasPrefix(address, "https://sys.openai"+ctxSeparator) {
		address = strings.TrimPrefix(strings.TrimPrefix(address, "https://"), "http://")
	}

	tool, ctx, err := toolNameAndCtxFromAddress(address)
	if err != nil {
		return Credential{}, err
	}

	return Credential{
		Context:  ctx,
		ToolName: tool,
		Type:     CredentialType(credType),
		Env:      env,
	}, nil
}

func toolNameWithCtx(toolName, credCtx string) string {
	return toolName + ctxSeparator + credCtx
}

func toolNameAndCtxFromAddress(address string) (string, string, error) {
	parts := strings.Split(address, ctxSeparator)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("error parsing tool name and context %q. Tool names cannot contain '%s'", address, ctxSeparator)
	}
	return parts[0], parts[1], nil
}
