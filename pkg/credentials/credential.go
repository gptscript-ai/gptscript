package credentials

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cli/cli/config/types"
)

type Credential struct {
	Context  string            `json:"context"`
	ToolName string            `json:"toolName"`
	Env      map[string]string `json:"env"`
}

func (c Credential) toDockerAuthConfig() (types.AuthConfig, error) {
	env, err := json.Marshal(c.Env)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      "gptscript", // Username is required, but not used
		Password:      string(env),
		ServerAddress: toolNameWithCtx(c.ToolName, c.Context),
	}, nil
}

func credentialFromDockerAuthConfig(authCfg types.AuthConfig) (Credential, error) {
	var env map[string]string
	if err := json.Unmarshal([]byte(authCfg.Password), &env); err != nil {
		return Credential{}, err
	}

	tool, ctx, err := toolNameAndCtxFromAddress(strings.TrimPrefix(authCfg.ServerAddress, "https://"))
	if err != nil {
		return Credential{}, err
	}

	return Credential{
		Context:  ctx,
		ToolName: tool,
		Env:      env,
	}, nil
}

func toolNameWithCtx(toolName, credCtx string) string {
	return toolName + "///" + credCtx
}

func toolNameAndCtxFromAddress(address string) (string, string, error) {
	parts := strings.Split(address, "///")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("error parsing tool name and context %q. Tool names cannot contain '///'", address)
	}
	return parts[0], parts[1], nil
}
