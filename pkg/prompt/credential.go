package prompt

import (
	"context"
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/tidwall/gjson"
)

func GetModelProviderCredential(ctx context.Context, credStore credentials.CredentialStore, credName, env, message string, envs []string) (string, error) {
	cred, exists, err := credStore.Get(ctx, credName)
	if err != nil {
		return "", err
	}

	var k string
	if exists {
		k = cred.Env[env]
	} else {
		// we know progress isn't used so pass as nil
		result, err := SysPrompt(ctx, envs, fmt.Sprintf(`{"message":"%s","fields":"key","sensitive":"true"}`, message), nil)
		if err != nil {
			return "", err
		}

		k = gjson.Get(result, "key").String()
		if err := credStore.Add(ctx, credentials.Credential{
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
