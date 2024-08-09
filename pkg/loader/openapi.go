package loader

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/openapi"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var toolNameRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// getOpenAPITools parses an OpenAPI definition and generates a set of tools from it.
// Each operation will become a tool definition.
// The tool's Instructions will be in the format "#!sys.openapi '{JSON Instructions}'",
// where the JSON Instructions are a JSON-serialized openapi.OperationInfo struct.
func getOpenAPITools(t *openapi3.T, defaultHost, source, targetToolName string) ([]types.Tool, error) {
	if os.Getenv("GPTSCRIPT_OPENAPI_REVAMP") == "true" {
		return getOpenAPIToolsRevamp(t, source, targetToolName)
	}

	if log.IsDebug() {
		start := time.Now()
		defer func() {
			log.Debugf("loaded openapi tools in %v", time.Since(start))
		}()
	}
	// Determine the default server.
	if len(t.Servers) == 0 {
		if defaultHost != "" {
			u, err := url.Parse(defaultHost)
			if err != nil {
				return nil, fmt.Errorf("invalid default host URL: %w", err)
			}
			u.Path = "/"
			t.Servers = []*openapi3.Server{{URL: u.String()}}
		} else {
			return nil, fmt.Errorf("no servers found in OpenAPI spec")
		}
	}
	defaultServer, err := parseServer(t.Servers[0])
	if err != nil {
		return nil, err
	}

	var globalSecurity []map[string]struct{}
	if t.Security != nil {
		for _, item := range t.Security {
			current := map[string]struct{}{}
			for name := range item {
				if scheme, ok := t.Components.SecuritySchemes[name]; ok && slices.Contains(openapi.GetSupportedSecurityTypes(), scheme.Value.Type) {
					current[name] = struct{}{}
				}
			}
			if len(current) > 0 {
				globalSecurity = append(globalSecurity, current)
			}
		}
	}

	// Generate a tool for each operation.
	var (
		toolNames    []string
		tools        []types.Tool
		operationNum = 1 // Each tool gets an operation number, beginning with 1
	)

	pathMap := t.Paths.Map()

	keys := make([]string, 0, len(pathMap))
	for k := range pathMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, pathString := range keys {
		pathObj := pathMap[pathString]
		// Handle path-level server override, if one exists
		pathServer := defaultServer
		if pathObj.Servers != nil && len(pathObj.Servers) > 0 {
			pathServer, err = parseServer(pathObj.Servers[0])
			if err != nil {
				return nil, err
			}
		}

		// Generate a tool for each operation in this path.
		operations := pathObj.Operations()
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)
	operations:
		for _, method := range methods {
			operation := operations[method]
			// Handle operation-level server override, if one exists
			operationServer := pathServer
			if operation.Servers != nil && len(*operation.Servers) > 0 {
				operationServer, err = parseServer((*operation.Servers)[0])
				if err != nil {
					return nil, err
				}
			}

			// Each operation can have a description and a summary. Use the Description if one exists,
			// otherwise us the summary.
			toolDesc := operation.Description
			if toolDesc == "" {
				toolDesc = operation.Summary
			}

			if len(toolDesc) > 1024 {
				toolDesc = toolDesc[:1024]
			}

			toolName := operation.OperationID
			if toolName == "" {
				// When there is no operation ID, we use the method + path as the tool name and remove all characters
				// except letters, numbers, underscores, and hyphens.
				toolName = toolNameRegex.ReplaceAllString(strings.ToLower(method)+strings.ReplaceAll(pathString, "/", "_"), "")
			}

			var (
				// auths are represented as a list of maps, where each map contains the names of the required security schemes.
				// Items within the same map are a logical AND. The maps themselves are a logical OR. For example:
				//	 security: # (A AND B) OR (C AND D)
				//   - A
				//     B
				//   - C
				//     D
				auths            []map[string]struct{}
				queryParameters  []openapi.Parameter
				pathParameters   []openapi.Parameter
				headerParameters []openapi.Parameter
				cookieParameters []openapi.Parameter
				bodyMIME         string
			)
			tool := types.Tool{
				ToolDef: types.ToolDef{
					Parameters: types.Parameters{
						Name:        toolName,
						Description: toolDesc,
						Arguments: &openapi3.Schema{
							Type:       &openapi3.Types{"object"},
							Properties: openapi3.Schemas{},
							Required:   []string{},
						},
					},
				},
				Source: types.ToolSource{
					// We need some concept of a line number in order for tools to have different IDs
					// So we basically just treat it as an "operation number" in this case
					LineNo: operationNum,
				},
			}

			// Handle query, path, and header parameters, based on the parameters for this operation
			// and the parameters for this path.
			for _, param := range append(operation.Parameters, pathObj.Parameters...) {
				arg := param.Value.Schema.Value

				if arg.Description == "" {
					arg.Description = param.Value.Description
				}

				// Add the new arg to the tool's arguments
				tool.Parameters.Arguments.Properties[param.Value.Name] = &openapi3.SchemaRef{Value: arg}

				// Check whether it is required
				if param.Value.Required {
					tool.Parameters.Arguments.Required = append(tool.Parameters.Arguments.Required, param.Value.Name)
				}

				// Add the parameter to the appropriate list for the tool's instructions
				p := openapi.Parameter{
					Name:    param.Value.Name,
					Style:   param.Value.Style,
					Explode: param.Value.Explode,
				}
				switch param.Value.In {
				case "query":
					queryParameters = append(queryParameters, p)
				case "path":
					pathParameters = append(pathParameters, p)
				case "header":
					headerParameters = append(headerParameters, p)
				case "cookie":
					cookieParameters = append(cookieParameters, p)
				}
			}

			// Handle the request body, if one exists
			if operation.RequestBody != nil {
				for mime, content := range operation.RequestBody.Value.Content {
					// Each MIME type needs to be handled individually, so we
					// keep a list of the ones we support.
					if !slices.Contains(openapi.GetSupportedMIMETypes(), mime) {
						continue
					}
					bodyMIME = mime

					arg := content.Schema.Value
					if arg.Description == "" {
						arg.Description = content.Schema.Value.Description
					}

					// Read Only can not be sent in the request body, so we remove it
					for key, property := range arg.Properties {
						if property.Value.ReadOnly {
							delete(arg.Properties, key)
						}
					}
					// Unfortunately, the request body doesn't contain any good descriptor for it,
					// so we just use "requestBodyContent" as the name of the arg.
					tool.Parameters.Arguments.Properties["requestBodyContent"] = &openapi3.SchemaRef{Value: arg}
					break
				}

				if bodyMIME == "" {
					// No supported MIME types found, so just skip this operation and move on.
					continue operations
				}
			}

			// See if there is any auth defined for this operation
			var noAuth bool
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
			var infos [][]openapi.SecurityInfo
		outer:
			for _, auth := range auths {
				var current []openapi.SecurityInfo
				for name := range auth {
					if scheme, ok := t.Components.SecuritySchemes[name]; ok {
						if !slices.Contains(openapi.GetSupportedSecurityTypes(), scheme.Value.Type) {
							// There is an unsupported type in this auth, so move on to the next one.
							continue outer
						}

						current = append(current, openapi.SecurityInfo{
							Type:       scheme.Value.Type,
							Name:       name,
							In:         scheme.Value.In,
							Scheme:     scheme.Value.Scheme,
							APIKeyName: scheme.Value.Name,
						})
					}
				}

				if len(current) > 0 {
					infos = append(infos, current)
				}
			}

			// OpenAI will get upset if we have an object schema with no properties,
			// so we just nil this out if there were no properties added.
			if len(tool.Arguments.Properties) == 0 {
				tool.Arguments = nil
			}

			var err error
			tool.Instructions, err = instructionString(operationServer, method, pathString, bodyMIME, queryParameters, pathParameters, headerParameters, cookieParameters, infos)
			if err != nil {
				return nil, err
			}

			if len(infos) > 0 {
				// Set up credential tools for the first set of infos.
				for _, info := range infos[0] {
					operationServerURL, err := url.Parse(operationServer)
					if err != nil {
						return nil, fmt.Errorf("failed to parse operation server URL: %w", err)
					}
					for _, cred := range info.GetCredentialToolStrings(operationServerURL.Hostname()) {
						tool.Credentials = append(tool.Credentials, cred)
					}
				}
			}

			// Register
			toolNames = append(toolNames, tool.Parameters.Name)
			tools = append(tools, tool)
			operationNum++
		}
	}

	// The first tool we generate is a special tool that just exports all the others.
	exportTool := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Description: fmt.Sprintf("This is a tool set for the %s OpenAPI spec", t.Info.Title),
				Export:      toolNames,
			},
		},
		Source: types.ToolSource{
			LineNo: 0,
		},
	}
	// Add it to the front of the slice.
	tools = append([]types.Tool{exportTool}, tools...)

	return tools, nil
}

func instructionString(server, method, path, bodyMIME string, queryParameters, pathParameters, headerParameters, cookieParameters []openapi.Parameter, infos [][]openapi.SecurityInfo) (string, error) {
	inst := openapi.OperationInfo{
		Server:          server,
		Path:            path,
		Method:          method,
		BodyContentMIME: bodyMIME,
		SecurityInfos:   infos,
		QueryParams:     queryParameters,
		PathParams:      pathParameters,
		HeaderParams:    headerParameters,
		CookieParams:    cookieParameters,
	}
	instBytes, err := json.Marshal(inst)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool instructions: %w", err)
	}

	return fmt.Sprintf("%s '%s'", types.OpenAPIPrefix, string(instBytes)), nil
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

func getOpenAPIToolsRevamp(t *openapi3.T, source, targetToolName string) ([]types.Tool, error) {
	if t == nil {
		return nil, fmt.Errorf("OpenAPI spec is nil")
	} else if t.Info == nil {
		return nil, fmt.Errorf("OpenAPI spec is missing info field")
	}

	if targetToolName == "" {
		targetToolName = openapi.NoFilter
	}

	list := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name:        types.ToolNormalizer("list-operations-" + t.Info.Title),
				Description: fmt.Sprintf("List available operations for %s. Each of these operations is an OpenAPI operation. Run this tool before you do anything else.", t.Info.Title),
			},
			Instructions: fmt.Sprintf("%s %s %s %s", types.OpenAPIPrefix, openapi.ListTool, source, targetToolName),
		},
		Source: types.ToolSource{
			LineNo: 0,
		},
	}

	getSchema := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name:        types.ToolNormalizer("get-schema-" + t.Info.Title),
				Description: fmt.Sprintf("Get the JSONSchema for the arguments for an operation for %s. You must do this before you run the operation.", t.Info.Title),
				Arguments: &openapi3.Schema{
					Type: &openapi3.Types{openapi3.TypeObject},
					Properties: openapi3.Schemas{
						"operation": {
							Value: &openapi3.Schema{
								Type:        &openapi3.Types{openapi3.TypeString},
								Title:       "operation",
								Description: "the name of the operation to get the schema for",
								Required:    []string{"operation"},
							},
						},
					},
				},
			},
			Instructions: fmt.Sprintf("%s %s %s %s", types.OpenAPIPrefix, openapi.GetSchemaTool, source, targetToolName),
		},
		Source: types.ToolSource{
			LineNo: 1,
		},
	}

	run := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name:        types.ToolNormalizer("run-operation-" + t.Info.Title),
				Description: fmt.Sprintf("Run an operation for %s. You MUST call %s for the operation before you use this tool.", t.Info.Title, openapi.GetSchemaTool),
				Arguments: &openapi3.Schema{
					Type: &openapi3.Types{openapi3.TypeObject},
					Properties: openapi3.Schemas{
						"operation": {
							Value: &openapi3.Schema{
								Type:        &openapi3.Types{openapi3.TypeString},
								Title:       "operation",
								Description: "the name of the operation to run",
								Required:    []string{"operation"},
							},
						},
						"args": {
							Value: &openapi3.Schema{
								Type:        &openapi3.Types{openapi3.TypeString},
								Title:       "args",
								Description: "the JSON string containing arguments; must match the JSONSchema for the operation",
								Required:    []string{"args"},
							},
						},
					},
				},
			},
			Instructions: fmt.Sprintf("%s %s %s %s", types.OpenAPIPrefix, openapi.RunTool, source, targetToolName),
		},
	}

	exportTool := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Export: []string{list.Parameters.Name, getSchema.Parameters.Name, run.Parameters.Name},
			},
		},
	}

	return []types.Tool{exportTool, list, getSchema, run}, nil
}
