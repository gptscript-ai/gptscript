package credentials

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
)

// ParseCredentialOverrides parses a string of credential overrides that the user provided as a command line arg.
// The format of credential overrides can be one of two things:
// cred1:ENV1,ENV2 (direct mapping of environment variables)
// cred1:ENV1=VALUE1,ENV2=VALUE2 (key-value pairs)
//
// This function turns it into a map[string]map[string]string like this:
//
//	{
//	  "cred1": {
//	    "ENV1": "VALUE1",
//	    "ENV2": "VALUE2",
//	  }
//	}
func ParseCredentialOverrides(overrides []string) (map[string]map[string]string, error) {
	credentialOverrides := make(map[string]map[string]string)

	for _, o := range overrides {
		credName, envs, found := strings.Cut(o, ":")
		if !found {
			return nil, fmt.Errorf("invalid credential override: %s", o)
		}
		envMap, ok := credentialOverrides[credName]
		if !ok {
			envMap = make(map[string]string)
		}
		for _, env := range strings.Split(envs, ",") {
			for _, env := range strings.Split(env, "|") {
				key, value, found := strings.Cut(env, "=")
				if !found {
					// User just passed an env var name as the key, so look up the value.
					value = os.Getenv(key)
				}
				envMap[key] = value
			}
		}
		credentialOverrides[credName] = envMap
	}

	return credentialOverrides, nil
}

type withOverride struct {
	target      CredentialStore
	credContext []string
	overrides   map[string]map[string]map[string]string
}

func (w withOverride) Get(ctx context.Context, toolName string) (*Credential, bool, error) {
	for _, credCtx := range w.credContext {
		overrides, ok := w.overrides[credCtx]
		if !ok {
			continue
		}
		override, ok := overrides[toolName]
		if !ok {
			continue
		}

		return &Credential{
			Context:  credCtx,
			ToolName: toolName,
			Type:     CredentialTypeTool,
			Env:      maps.Clone(override),
		}, true, nil
	}

	return w.target.Get(ctx, toolName)
}

func (w withOverride) Add(ctx context.Context, cred Credential) error {
	for _, credCtx := range w.credContext {
		if override, ok := w.overrides[credCtx]; ok {
			if _, ok := override[cred.ToolName]; ok {
				return fmt.Errorf("cannot add credential with context %q and tool %q because it is statically configure", cred.Context, cred.ToolName)
			}
		}
	}
	return w.target.Add(ctx, cred)
}

func (w withOverride) Refresh(ctx context.Context, cred Credential) error {
	if override, ok := w.overrides[cred.Context]; ok {
		if _, ok := override[cred.ToolName]; ok {
			return nil
		}
	}
	return w.target.Refresh(ctx, cred)
}

func (w withOverride) Remove(ctx context.Context, toolName string) error {
	for _, credCtx := range w.credContext {
		if override, ok := w.overrides[credCtx]; ok {
			if _, ok := override[toolName]; ok {
				return fmt.Errorf("cannot remove credential with context %q and tool %q because it is statically configure", credCtx, toolName)
			}
		}
	}
	return w.target.Remove(ctx, toolName)
}

func (w withOverride) List(ctx context.Context) ([]Credential, error) {
	creds, err := w.target.List(ctx)
	if err != nil {
		return nil, err
	}

	added := make(map[string]map[string]bool)
	for i, cred := range creds {
		if override, ok := w.overrides[cred.Context]; ok {
			if _, ok := override[cred.ToolName]; ok {
				creds[i].Type = CredentialTypeTool
				creds[i].Env = maps.Clone(override[cred.ToolName])
			}
		}
		tools, ok := added[cred.Context]
		if !ok {
			tools = make(map[string]bool)
		}
		tools[cred.ToolName] = true
		added[cred.Context] = tools
	}

	for _, credCtx := range w.credContext {
		tools := w.overrides[credCtx]
		for toolName := range tools {
			if _, ok := added[credCtx][toolName]; ok {
				continue
			}
			creds = append(creds, Credential{
				Context:  credCtx,
				ToolName: toolName,
				Type:     CredentialTypeTool,
				Env:      maps.Clone(tools[toolName]),
			})
		}
	}

	return creds, nil
}
