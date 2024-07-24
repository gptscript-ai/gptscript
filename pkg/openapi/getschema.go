package openapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type Parameter struct {
	Name    string `json:"name"`
	Style   string `json:"style"`
	Explode *bool  `json:"explode"`
}

type OperationInfo struct {
	Server          string           `json:"server"`
	Path            string           `json:"path"`
	Method          string           `json:"method"`
	BodyContentMIME string           `json:"bodyContentMIME"`
	SecurityInfos   [][]SecurityInfo `json:"securityInfos"`
	QueryParams     []Parameter      `json:"queryParameters"`
	PathParams      []Parameter      `json:"pathParameters"`
	HeaderParams    []Parameter      `json:"headerParameters"`
	CookieParams    []Parameter      `json:"cookieParameters"`
}

var (
	SupportedMIMETypes     = []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}
	SupportedSecurityTypes = []string{"apiKey", "http"}
)

// GetSchema returns the JSONSchema and OperationInfo for a particular OpenAPI operation.
// Return values in order: JSONSchema (string), OperationInfo, found (bool), error.
func GetSchema(operationID, defaultHost string, t *openapi3.T) (string, OperationInfo, bool, error) {
	// We basically want to extract all the information that we need for the HTTP request,
	// like we do in GPTScript.
	arguments := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: openapi3.Schemas{},
		Required:   []string{},
	}

	info := OperationInfo{}

	// Determine the default server.
	var (
		defaultServer = defaultHost
		err           error
	)
	if len(t.Servers) > 0 {
		defaultServer, err = parseServer(t.Servers[0])
		if err != nil {
			return "", OperationInfo{}, false, err
		}
	}

	var globalSecurity []map[string]struct{}
	if t.Security != nil {
		for _, item := range t.Security {
			current := map[string]struct{}{}
			for name := range item {
				if scheme, ok := t.Components.SecuritySchemes[name]; ok && slices.Contains(SupportedSecurityTypes, scheme.Value.Type) {
					current[name] = struct{}{}
				}
			}
			if len(current) > 0 {
				globalSecurity = append(globalSecurity, current)
			}
		}
	}

	for path, pathItem := range t.Paths.Map() {
		// Handle path-level server override, if one exists.
		pathServer := defaultServer
		if pathItem.Servers != nil && len(pathItem.Servers) > 0 {
			pathServer, err = parseServer(pathItem.Servers[0])
			if err != nil {
				return "", OperationInfo{}, false, err
			}
		}

		for method, operation := range pathItem.Operations() {
			if operation.OperationID == operationID {
				// Handle operation-level server override, if one exists.
				operationServer := pathServer
				if operation.Servers != nil && len(*operation.Servers) > 0 {
					operationServer, err = parseServer((*operation.Servers)[0])
					if err != nil {
						return "", OperationInfo{}, false, err
					}
				}

				info.Server = operationServer
				info.Path = path
				info.Method = method

				// We found our operation. Now we need to process it and build the arguments.
				// Handle query, path, header, and cookie parameters first.
				for _, param := range append(operation.Parameters, pathItem.Parameters...) {
					removeRefs(param.Value.Schema)
					arg := param.Value.Schema.Value

					if arg.Description == "" {
						arg.Description = param.Value.Description
					}

					// Store the arg
					arguments.Properties[param.Value.Name] = &openapi3.SchemaRef{Value: arg}

					// Check whether it is required
					if param.Value.Required {
						arguments.Required = append(arguments.Required, param.Value.Name)
					}

					// Save the parameter to the correct set of params.
					p := Parameter{
						Name:    param.Value.Name,
						Style:   param.Value.Style,
						Explode: param.Value.Explode,
					}
					switch param.Value.In {
					case "query":
						info.QueryParams = append(info.QueryParams, p)
					case "path":
						info.PathParams = append(info.PathParams, p)
					case "header":
						info.HeaderParams = append(info.HeaderParams, p)
					case "cookie":
						info.CookieParams = append(info.CookieParams, p)
					}
				}

				// Next, handle the request body, if one exists.
				if operation.RequestBody != nil {
					for mime, content := range operation.RequestBody.Value.Content {
						// Each MIME type needs to be handled individually, so we keep a list of the ones we support.
						if !slices.Contains(SupportedMIMETypes, mime) {
							continue
						}
						info.BodyContentMIME = mime

						removeRefs(content.Schema)

						arg := content.Schema.Value
						if arg.Description == "" {
							arg.Description = content.Schema.Value.Description
						}

						// Read Only cannot be sent in the request body, so we remove it
						for key, property := range arg.Properties {
							if property.Value.ReadOnly {
								delete(arg.Properties, key)
							}
						}

						// Unfortunately, the request body doesn't contain any good descriptor for it,
						// so we just use "requestBodyContent" as the name of the arg.
						arguments.Properties["requestBodyContent"] = &openapi3.SchemaRef{Value: arg}
						arguments.Required = append(arguments.Required, "requestBodyContent")
						break
					}

					if info.BodyContentMIME == "" {
						return "", OperationInfo{}, false, fmt.Errorf("no supported MIME type found for request body in operation %s", operationID)
					}
				}

				// See if there is any auth defined for this operation
				var (
					noAuth bool
					auths  []map[string]struct{}
				)
				if operation.Security != nil {
					if len(*operation.Security) == 0 {
						noAuth = true
					}
					for _, req := range *operation.Security {
						current := map[string]struct{}{}
						for name := range req {
							current[name] = struct{}{}
						}
						if len(current) > 0 {
							auths = append(auths, current)
						}
					}
				}

				// Use the global security if it was not overridden for this operation
				if !noAuth && len(auths) == 0 {
					auths = append(auths, globalSecurity...)
				}

				// For each set of auths, turn them into SecurityInfos, and drop ones that contain unsupported types.
			outer:
				for _, auth := range auths {
					var current []SecurityInfo
					for name := range auth {
						if scheme, ok := t.Components.SecuritySchemes[name]; ok {
							if !slices.Contains(SupportedSecurityTypes, scheme.Value.Type) {
								// There is an unsupported type in this auth, so move on to the next one.
								continue outer
							}

							current = append(current, SecurityInfo{
								Type:       scheme.Value.Type,
								Name:       name,
								In:         scheme.Value.In,
								Scheme:     scheme.Value.Scheme,
								APIKeyName: scheme.Value.Name,
							})
						}
					}

					if len(current) > 0 {
						info.SecurityInfos = append(info.SecurityInfos, current)
					}
				}

				argumentsJSON, err := json.MarshalIndent(arguments, "", "    ")
				if err != nil {
					return "", OperationInfo{}, false, err
				}
				return string(argumentsJSON), info, true, nil
			}
		}
	}

	return "", OperationInfo{}, false, nil
}

func parseServer(server *openapi3.Server) (string, error) {
	s := server.URL
	for name, variable := range server.Variables {
		if variable == nil {
			continue
		}

		if variable.Default != "" {
			s = strings.Replace(s, "{"+name+"}", variable.Default, 1)
		} else if len(variable.Enum) > 0 {
			s = strings.Replace(s, "{"+name+"}", variable.Enum[0], 1)
		}
	}

	if !strings.HasPrefix(s, "http") {
		return "", fmt.Errorf("invalid server URL: %s (must use HTTP or HTTPS; relative URLs not supported)", s)
	}
	return s, nil
}

func removeRefs(r *openapi3.SchemaRef) {
	if r == nil {
		return
	}

	r.Ref = ""
	r.Value.Discriminator = nil // Discriminators are not very useful and can junk up the schema.

	for i := range r.Value.OneOf {
		removeRefs(r.Value.OneOf[i])
	}
	for i := range r.Value.AnyOf {
		removeRefs(r.Value.AnyOf[i])
	}
	for i := range r.Value.AllOf {
		removeRefs(r.Value.AllOf[i])
	}
	removeRefs(r.Value.Not)
	removeRefs(r.Value.Items)

	for i := range r.Value.Properties {
		removeRefs(r.Value.Properties[i])
	}
}
