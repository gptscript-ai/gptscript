package loader

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func getOpenAPITools(t *openapi3.T) ([]types.Tool, error) {
	if len(t.Servers) == 0 {
		return nil, fmt.Errorf("no servers found in OpenAPI spec")
	}
	defaultServer, err := parseServer(t.Servers[0])
	if err != nil {
		return nil, err
	}

	// TODO - support security schemes?
	var (
		toolNames    []string
		tools        []types.Tool
		operationNum = 1
	)
	for pathString, pathObj := range t.Paths.Map() {
		// Handle path-level server override, if one exists
		pathServer := defaultServer
		if pathObj.Servers != nil && len(pathObj.Servers) > 0 {
			pathServer, err = parseServer(pathObj.Servers[0])
			if err != nil {
				return nil, err
			}
		}

		for method, operation := range pathObj.Operations() {
			// Handle operation-level server override, if one exists
			operationServer := pathServer
			if operation.Servers != nil && len(*operation.Servers) > 0 {
				operationServer, err = parseServer((*operation.Servers)[0])
				if err != nil {
					return nil, err
				}
			}

			toolDesc := operation.Description
			if toolDesc == "" {
				toolDesc = operation.Summary
			}

			var (
				queryParameters  []engine.Parameter
				pathParameters   []engine.Parameter
				headerParameters []engine.Parameter
				cookieParameters []engine.Parameter
				bodyMIME         string
			)
			tool := types.Tool{
				Parameters: types.Parameters{
					Name:        operation.OperationID,
					Description: toolDesc,
					Arguments: &types.JSONSchema{
						Type:       "object",
						Properties: map[string]types.JSONSchema{},
						Required:   []string{},
						Defs:       map[string]types.JSONSchema{},
					},
				},
				Source: types.ToolSource{
					// We need some concept of a line number in order for tools to have different IDs
					// So we basically just treat it as an "operation number" in this case
					LineNo: operationNum,
				},
			}

			toolNames = append(toolNames, tool.Parameters.Name)

			// Handle query, path, and header parameters
			for _, param := range operation.Parameters {
				// Marshal the param's schema to a string, and then into our JSONSchema struct
				arg := types.JSONSchema{}
				raw, err := param.Value.Schema.Value.MarshalJSON()
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(raw, &arg); err != nil {
					return nil, err
				}
				if arg.Description == "" {
					arg.Description = param.Value.Description
				}
				tool.Parameters.Arguments.Properties[param.Value.Name] = arg

				// Check whether it is required
				if param.Value.Required {
					tool.Parameters.Arguments.Required = append(tool.Parameters.Arguments.Required, param.Value.Name)
				}

				// Add the parameter to the appropriate list
				p := engine.Parameter{
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

			if operation.RequestBody != nil {
				for mime, content := range operation.RequestBody.Value.Content {
					if !slices.Contains(engine.SupportedMIMETypes, mime) {
						continue
					}
					bodyMIME = mime

					arg := types.JSONSchema{}
					raw, err := content.Schema.Value.MarshalJSON()
					if err != nil {
						return nil, err
					}
					if err := json.Unmarshal(raw, &arg); err != nil {
						return nil, err
					}
					if arg.Description == "" {
						arg.Description = content.Schema.Value.Description
					}

					tool.Parameters.Arguments.Properties["requestBodyContent"] = arg
					break
				}

				if bodyMIME == "" {
					return nil, fmt.Errorf("no supported MIME types found for request body in operation %s", operation.OperationID)
				}
			}

			var err error
			tool.Instructions, err = instructionString(operationServer, method, pathString, bodyMIME, queryParameters, pathParameters, headerParameters, cookieParameters)
			if err != nil {
				return nil, err
			}

			tools = append(tools, tool)
			operationNum++
		}
	}

	exportTool := types.Tool{
		Parameters: types.Parameters{
			Description: fmt.Sprintf("This is a tool set for the %s OpenAPI spec", t.Info.Title),
			Export:      toolNames,
		},
		Source: types.ToolSource{
			LineNo: 0,
		},
	}
	tools = append([]types.Tool{exportTool}, tools...)

	// TODO - get defs?

	return tools, nil
}

func instructionString(server, method, path, bodyMIME string, queryParameters, pathParameters, headerParameters, cookieParameters []engine.Parameter) (string, error) {
	inst := engine.OpenAPIInstructions{
		Server:           server,
		Path:             path,
		Method:           method,
		BodyContentMIME:  bodyMIME,
		QueryParameters:  queryParameters,
		PathParameters:   pathParameters,
		HeaderParameters: headerParameters,
		CookieParameters: cookieParameters,
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
		return "", fmt.Errorf("invalid server URL: %s (must use HTTP or HTTPs, relative URLs not supported)", s)
	}
	return s, nil
}
