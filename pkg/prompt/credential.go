package prompt

import (
	"context"
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/tidwall/gjson"
)

func GetModelProviderCredential(ctx context.Context, credName, env, message, credCtx string, envs []string, cliCfg *config.CLIConfig) (string, error) {
	store, err := credentials.NewStore(cliCfg, credCtx)
	if err != nil {
		return "", err
	}

	cred, exists, err := store.Get(credName)
	if err != nil {
		return "", err
	}

	var k string
	if exists {
		k = cred.Env[env]
	} else {
		result, err := SysPrompt(ctx, envs, fmt.Sprintf(`{"message":"%s","fields":"key","sensitive":"true"}`, message))
		if err != nil {
			return "", err
		}

		k = gjson.Get(result, "key").String()
		if err := store.Add(credentials.Credential{
			ToolName: credName,
			Type:     credentials.CredentialTypeModelProvider,
			Env: map[string]string{
				env: k,
			},
		}); err != nil {
			return "", err
		}
		log.Infof("Saved API key as credential %s", credName)
	}

	return k, nil
}
