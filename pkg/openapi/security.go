package openapi

import (
	"fmt"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/env"
)

// A SecurityInfo represents a security scheme in OpenAPI.
type SecurityInfo struct {
	Name       string `json:"name"`       // name as defined in the security schemes
	Type       string `json:"type"`       // http or apiKey
	Scheme     string `json:"scheme"`     // bearer or basic, for type==http
	APIKeyName string `json:"apiKeyName"` // name of the API key, for type==apiKey
	In         string `json:"in"`         // header, query, or cookie, for type==apiKey
}

func (i SecurityInfo) GetCredentialToolStrings(hostname string) []string {
	vars := i.getCredentialNamesAndEnvVars(hostname)
	var tools []string

	ctool := env.VarOrDefault("GPTSCRIPT_OPENAPI_CREDENTIAL_TOOL", "github.com/gptscript-ai/credential")

	for cred, v := range vars {
		field := "value"
		switch i.Type {
		case "apiKey":
			field = i.APIKeyName
		case "http":
			if i.Scheme == "bearer" {
				field = "bearer token"
			} else {
				if strings.Contains(v, "PASSWORD") {
					field = "password"
				} else {
					field = "username"
				}
			}
		}

		tools = append(tools, fmt.Sprintf("%s as %s with %s as env and %q as message and %q as field",
			ctool, cred, v, "Please provide a value for the "+v+" environment variable", field))
	}
	return tools
}

func (i SecurityInfo) getCredentialNamesAndEnvVars(hostname string) map[string]string {
	if i.Type == "http" && i.Scheme == "basic" {
		return map[string]string{
			hostname + i.Name + "Username": "GPTSCRIPT_" + env.ToEnvLike(hostname) + "_" + env.ToEnvLike(i.Name) + "_USERNAME",
			hostname + i.Name + "Password": "GPTSCRIPT_" + env.ToEnvLike(hostname) + "_" + env.ToEnvLike(i.Name) + "_PASSWORD",
		}
	}
	return map[string]string{
		hostname + i.Name: "GPTSCRIPT_" + env.ToEnvLike(hostname) + "_" + env.ToEnvLike(i.Name),
	}
}
