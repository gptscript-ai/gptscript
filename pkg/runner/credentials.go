package runner

import (
	"fmt"
	"os"
	"strings"
)

// parseCredentialOverrides parses a string of credential overrides that the user provided as a command line arg.
// The format of credential overrides can be one of three things:
// tool1:ENV1,ENV2;tool2:ENV1,ENV2 (direct mapping of environment variables)
// tool1:ENV1=VALUE1,ENV2=VALUE2;tool2:ENV1=VALUE1,ENV2=VALUE2 (key-value pairs)
// tool1:ENV1->OTHER_ENV1,ENV2->OTHER_ENV2;tool2:ENV1->OTHER_ENV1,ENV2->OTHER_ENV2 (mapping to other environment variables)
//
// This function turns it into a map[string]map[string]string like this:
//
//	{
//	  "tool1": {
//	    "ENV1": "VALUE1",
//	    "ENV2": "VALUE2",
//	  },
//	  "tool2": {
//	    "ENV1": "VALUE1",
//	    "ENV2": "VALUE2",
//	  },
//	}
func parseCredentialOverrides(override string) (map[string]map[string]string, error) {
	credentialOverrides := make(map[string]map[string]string)

	for _, o := range strings.Split(override, ";") {
		toolName, envs, found := strings.Cut(o, ":")
		if !found {
			return nil, fmt.Errorf("invalid credential override: %s", o)
		}
		envMap := make(map[string]string)
		for _, env := range strings.Split(envs, ",") {
			key, value, found := strings.Cut(env, "=")
			if !found {
				var envVar string
				key, envVar, found = strings.Cut(env, "->")
				if found {
					// User did a mapping of key -> other env var, so look up the value.
					value = os.Getenv(envVar)
				} else {
					// User just passed an env var name as the key, so look up the value.
					value = os.Getenv(key)
				}
			}
			envMap[key] = value
		}
		credentialOverrides[toolName] = envMap
	}

	return credentialOverrides, nil
}
