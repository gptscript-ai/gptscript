package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/openapi"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/tidwall/gjson"
)

func (e *Engine) runOpenAPIRevamp(tool types.Tool, input string) (*Return, error) {
	envMap := make(map[string]string, len(e.Env))
	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	_, inst, _ := strings.Cut(tool.Instructions, types.OpenAPIPrefix+" ")
	args := strings.Fields(inst)

	if len(args) != 3 {
		return nil, fmt.Errorf("expected 3 arguments to %s", types.OpenAPIPrefix)
	}

	command := args[0]
	source := args[1]
	filter := args[2]

	var res *Return
	switch command {
	case openapi.ListTool:
		t, err := openapi.Load(source)
		if err != nil {
			return nil, fmt.Errorf("failed to load OpenAPI file %s: %w", source, err)
		}

		opList, err := openapi.List(t, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to list operations: %w", err)
		}

		opListJSON, err := json.MarshalIndent(opList, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal operation list: %w", err)
		}

		res = &Return{
			Result: ptr(string(opListJSON)),
		}
	case openapi.GetSchemaTool:
		operation := gjson.Get(input, "operation").String()

		if filter != "" && filter != openapi.NoFilter {
			match, err := openapi.MatchFilters(strings.Split(filter, "|"), operation)
			if err != nil {
				return nil, err
			} else if !match {
				// Report to the LLM that the operation was not found
				return &Return{
					Result: ptr(fmt.Sprintf("operation %s not found", operation)),
				}, nil
			}
		}

		t, err := openapi.Load(source)
		if err != nil {
			return nil, fmt.Errorf("failed to load OpenAPI file %s: %w", source, err)
		}

		var defaultHost string
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			u, err := url.Parse(source)
			if err != nil {
				return nil, fmt.Errorf("failed to parse server URL %s: %w", source, err)
			}
			defaultHost = u.Scheme + "://" + u.Hostname()
		}

		schema, _, found, err := openapi.GetSchema(operation, defaultHost, t)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema: %w", err)
		}
		if !found {
			// Report to the LLM that the operation was not found
			return &Return{
				Result: ptr(fmt.Sprintf("operation %s not found", operation)),
			}, nil
		}

		schemaJSON, err := json.MarshalIndent(schema, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal schema: %w", err)
		}

		res = &Return{
			Result: ptr(string(schemaJSON)),
		}
	case openapi.RunTool:
		operation := gjson.Get(input, "operation").String()
		args := gjson.Get(input, "args").String()

		if filter != "" && filter != openapi.NoFilter {
			match, err := openapi.MatchFilters(strings.Split(filter, "|"), operation)
			if err != nil {
				return nil, err
			} else if !match {
				// Report to the LLM that the operation was not found
				return &Return{
					Result: ptr(fmt.Sprintf("operation %s not found", operation)),
				}, nil
			}
		}

		t, err := openapi.Load(source)
		if err != nil {
			return nil, fmt.Errorf("failed to load OpenAPI file %s: %w", source, err)
		}

		var defaultHost string
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			u, err := url.Parse(source)
			if err != nil {
				return nil, fmt.Errorf("failed to parse server URL %s: %w", source, err)
			}
			defaultHost = u.Scheme + "://" + u.Hostname()
		}

		result, found, err := openapi.Run(operation, defaultHost, args, t, e.Env)
		if err != nil {
			return nil, fmt.Errorf("failed to run operation %s: %w", operation, err)
		} else if !found {
			// Report to the LLM that the operation was not found
			return &Return{
				Result: ptr(fmt.Sprintf("operation %s not found", operation)),
			}, nil
		}

		res = &Return{
			Result: ptr(result),
		}
	}

	return res, nil
}

// runOpenAPI runs a tool that was generated from an OpenAPI definition.
// The tool itself will have instructions regarding the HTTP request that needs to be made.
// The tools Instructions field will be in the format "#!sys.openapi '{Instructions JSON}'",
// where {Instructions JSON} is a JSON string of type OpenAPIInstructions.
func (e *Engine) runOpenAPI(tool types.Tool, input string) (*Return, error) {
	if os.Getenv("GPTSCRIPT_OPENAPI_REVAMP") == "true" {
		return e.runOpenAPIRevamp(tool, input)
	}

	envMap := map[string]string{}

	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	// Extract the instructions from the tool to determine server, path, method, etc.
	var instructions openapi.OperationInfo
	_, inst, _ := strings.Cut(tool.Instructions, types.OpenAPIPrefix+" ")
	inst = strings.TrimPrefix(inst, "'")
	inst = strings.TrimSuffix(inst, "'")
	if err := json.Unmarshal([]byte(inst), &instructions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool instructions: %w", err)
	}

	// Handle path parameters
	instructions.Path = openapi.HandlePathParameters(instructions.Path, instructions.PathParams, input)

	// Parse the URL
	path, err := url.JoinPath(instructions.Server, instructions.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to join server and path: %w", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %s: %w", instructions.Server+instructions.Path, err)
	}

	// Set up the request
	req, err := http.NewRequest(instructions.Method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Check for authentication (only if using HTTPS or localhost)
	if u.Scheme == "https" || u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
		if len(instructions.SecurityInfos) > 0 {
			if err := openapi.HandleAuths(req, envMap, instructions.SecurityInfos); err != nil {
				return nil, fmt.Errorf("error setting up authentication: %w", err)
			}
		}

		// If there is a bearer token set for the whole server, and no Authorization header has been defined, use it.
		if token, ok := envMap["GPTSCRIPT_"+env.ToEnvLike(u.Hostname())+"_BEARER_TOKEN"]; ok {
			if req.Header.Get("Authorization") == "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
		}
	}

	// Handle query parameters
	req.URL.RawQuery = openapi.HandleQueryParameters(req.URL.Query(), instructions.QueryParams, input).Encode()

	// Handle header and cookie parameters
	openapi.HandleHeaderParameters(req, instructions.HeaderParams, input)
	openapi.HandleCookieParameters(req, instructions.CookieParams, input)

	// Handle request body
	if instructions.BodyContentMIME != "" {
		res := gjson.Get(input, "requestBodyContent")
		var body bytes.Buffer
		switch instructions.BodyContentMIME {
		case "application/json":
			var reqBody interface{}

			reqBody = struct{}{}
			if res.Exists() {
				reqBody = res.Value()
			}
			if err := json.NewEncoder(&body).Encode(reqBody); err != nil {
				return nil, fmt.Errorf("failed to encode JSON: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

		case "text/plain":
			reqBody := ""
			if res.Exists() {
				reqBody = res.String()
			}
			body.WriteString(reqBody)

			req.Header.Set("Content-Type", "text/plain")

		case "multipart/form-data":
			multiPartWriter := multipart.NewWriter(&body)
			req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
			if res.Exists() && res.IsObject() {
				for k, v := range res.Map() {
					if err := multiPartWriter.WriteField(k, v.String()); err != nil {
						return nil, fmt.Errorf("failed to write multipart field: %w", err)
					}
				}
			} else {
				return nil, fmt.Errorf("multipart/form-data requires an object as the requestBodyContent")
			}
			if err := multiPartWriter.Close(); err != nil {
				return nil, fmt.Errorf("failed to close multipart writer: %w", err)
			}

		default:
			return nil, fmt.Errorf("unsupported MIME type: %s", instructions.BodyContentMIME)
		}
		req.Body = io.NopCloser(&body)
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	resultStr := string(result)

	return &Return{
		Result: &resultStr,
	}, nil
}

func ptr[T any](t T) *T {
	return &t
}
