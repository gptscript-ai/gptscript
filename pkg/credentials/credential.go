package credentials

import (
	"encoding/json"

	"github.com/docker/cli/cli/config/types"
)

type Credential struct {
	ToolID string            `json:"toolID"`
	Env    map[string]string `json:"env"`
}

func (c Credential) toDockerAuthConfig() (types.AuthConfig, error) {
	env, err := json.Marshal(c.Env)
	if err != nil {
		return types.AuthConfig{}, err
	}

	return types.AuthConfig{
		Username:      "gptscript", // this field doesn't matter, but it needs to be set
		Password:      string(env),
		ServerAddress: c.ToolID,
	}, nil
}

func credentialFromDockerAuthConfig(authCfg types.AuthConfig) (Credential, error) {
	var env map[string]string
	if err := json.Unmarshal([]byte(authCfg.Password), &env); err != nil {
		return Credential{}, err
	}

	return Credential{
		ToolID: authCfg.ServerAddress,
		Env:    env,
	}, nil
}
