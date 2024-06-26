package credentials

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/docker/cli/cli/config/types"
)

type CredentialType string

const (
	ctxSeparator                               = "///"
	CredentialTypeTool          CredentialType = "tool"
	CredentialTypeModelProvider CredentialType = "modelProvider"
	ExistingCredential                         = "GPTSCRIPT_EXISTING_CREDENTIAL"
)

type Credential struct {
	Context      string            `json:"context"`
	ToolName     string            `json:"toolName"`
	Type         CredentialType    `json:"type"`
	Env          map[string]string `json:"env"`
	ExpiresAt    *time.Time        `json:"expiresAt"`
	RefreshToken string            `json:"refreshToken"`
}

func (c Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

func (c Credential) toDockerAuthConfig() (types.AuthConfig, error) {
	cred, err := json.Marshal(c)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      string(c.Type),
		Password:      string(cred),
		ServerAddress: toolNameWithCtx(c.ToolName, c.Context),
	}, nil
}

func credentialFromDockerAuthConfig(authCfg types.AuthConfig) (Credential, error) {
	var cred Credential
	if authCfg.Password != "" {
		if err := json.Unmarshal([]byte(authCfg.Password), &cred); err != nil || len(cred.Env) == 0 {
			// Legacy: try unmarshalling into just an env map
			var env map[string]string
			if err := json.Unmarshal([]byte(authCfg.Password), &env); err != nil {
				return Credential{}, err
			}
			cred.Env = env
		}
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
		Context:      ctx,
		ToolName:     tool,
		Type:         CredentialType(credType),
		Env:          cred.Env,
		ExpiresAt:    cred.ExpiresAt,
		RefreshToken: cred.RefreshToken,
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
