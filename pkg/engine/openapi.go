package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/maps"
)

var (
	SupportedMIMETypes     = []string{"application/json", "text/plain", "multipart/form-data"}
	SupportedSecurityTypes = []string{"apiKey", "http"}
)

type Parameter struct {
	Name    string `json:"name"`
	Style   string `json:"style"`
	Explode *bool  `json:"explode"`
}

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

		tools = append(tools, fmt.Sprintf("github.com/gptscript-ai/credential as %s with %s as env and %q as message and %q as field",
			cred, v, "Please provide a value for the "+v+" environment variable", field))
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

type OpenAPIInstructions struct {
	Server           string           `json:"server"`
	Path             string           `json:"path"`
	Method           string           `json:"method"`
	BodyContentMIME  string           `json:"bodyContentMIME"`
	SecurityInfos    [][]SecurityInfo `json:"apiKeyInfos"`
	QueryParameters  []Parameter      `json:"queryParameters"`
	PathParameters   []Parameter      `json:"pathParameters"`
	HeaderParameters []Parameter      `json:"headerParameters"`
	CookieParameters []Parameter      `json:"cookieParameters"`
}

// runOpenAPI runs a tool that was generated from an OpenAPI definition.
// The tool itself will have instructions regarding the HTTP request that needs to be made.
// The tools Instructions field will be in the format "#!sys.openapi '{Instructions JSON}'",
// where {Instructions JSON} is a JSON string of type OpenAPIInstructions.
func (e *Engine) runOpenAPI(tool types.Tool, input string) (*Return, error) {
	envMap := map[string]string{}

	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	// Extract the instructions from the tool to determine server, path, method, etc.
	var instructions OpenAPIInstructions
	_, inst, _ := strings.Cut(tool.Instructions, types.OpenAPIPrefix+" ")
	inst = strings.TrimPrefix(inst, "'")
	inst = strings.TrimSuffix(inst, "'")
	if err := json.Unmarshal([]byte(inst), &instructions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool instructions: %w", err)
	}

	// Handle path parameters
	instructions.Path = handlePathParameters(instructions.Path, instructions.PathParameters, input)

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
			if err := handleAuths(req, envMap, instructions.SecurityInfos); err != nil {
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
	req.URL.RawQuery = handleQueryParameters(req.URL.Query(), instructions.QueryParameters, input).Encode()

	// Handle header and cookie parameters
	handleHeaderParameters(req, instructions.HeaderParameters, input)
	handleCookieParameters(req, instructions.CookieParameters, input)

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

// handleAuths will set up the request with the necessary authentication information.
// A set of sets of SecurityInfo is passed in, where each represents a possible set of security options.
func handleAuths(req *http.Request, envMap map[string]string, infoSets [][]SecurityInfo) error {
	var missingVariables [][]string

	// We need to find a set of infos where we have all the needed environment variables.
	for _, infoSet := range infoSets {
		var missing []string // Keep track of any missing environment variables
		for _, info := range infoSet {
			vars := info.getCredentialNamesAndEnvVars(req.URL.Hostname())

			for _, envName := range vars {
				if _, ok := envMap[envName]; !ok {
					missing = append(missing, envName)
				}
			}
		}
		if len(missing) > 0 {
			missingVariables = append(missingVariables, missing)
			continue
		}

		// We're using this info set, because no environment variables were missing.
		// Set up the request as needed.
		for _, info := range infoSet {
			envNames := maps.Values(info.getCredentialNamesAndEnvVars(req.URL.Hostname()))
			switch info.Type {
			case "apiKey":
				switch info.In {
				case "header":
					req.Header.Set(info.APIKeyName, envMap[envNames[0]])
				case "query":
					v := url.Values{}
					v.Add(info.APIKeyName, envMap[envNames[0]])
					req.URL.RawQuery = v.Encode()
				case "cookie":
					req.AddCookie(&http.Cookie{
						Name:  info.APIKeyName,
						Value: envMap[envNames[0]],
					})
				}
			case "http":
				switch info.Scheme {
				case "bearer":
					req.Header.Set("Authorization", "Bearer "+envMap[envNames[0]])
				case "basic":
					req.SetBasicAuth(envMap[envNames[0]], envMap[envNames[1]])
				}
			}
		}
		return nil
	}

	return fmt.Errorf("did not find the needed environment variables for any of the security options. "+
		"At least one of these sets of environment variables must be provided: %v", missingVariables)
}

// handleQueryParameters extracts each query parameter from the input JSON and adds it to the URL query.
func handleQueryParameters(q url.Values, params []Parameter, input string) url.Values {
	for _, param := range params {
		res := gjson.Get(input, param.Name)
		if res.Exists() {
			// If it's an array or object, handle the serialization style
			if res.IsArray() {
				switch param.Style {
				case "form", "": // form is the default style for query parameters
					if param.Explode == nil || *param.Explode { // default is to explode
						for _, item := range res.Array() {
							q.Add(param.Name, item.String())
						}
					} else {
						var strs []string
						for _, item := range res.Array() {
							strs = append(strs, item.String())
						}
						q.Add(param.Name, strings.Join(strs, ","))
					}
				case "spaceDelimited":
					if param.Explode == nil || *param.Explode {
						for _, item := range res.Array() {
							q.Add(param.Name, item.String())
						}
					} else {
						var strs []string
						for _, item := range res.Array() {
							strs = append(strs, item.String())
						}
						q.Add(param.Name, strings.Join(strs, " "))
					}
				case "pipeDelimited":
					if param.Explode == nil || *param.Explode {
						for _, item := range res.Array() {
							q.Add(param.Name, item.String())
						}
					} else {
						var strs []string
						for _, item := range res.Array() {
							strs = append(strs, item.String())
						}
						q.Add(param.Name, strings.Join(strs, "|"))
					}
				}
			} else if res.IsObject() {
				switch param.Style {
				case "form", "": // form is the default style for query parameters
					if param.Explode == nil || *param.Explode { // default is to explode
						for k, v := range res.Map() {
							q.Add(k, v.String())
						}
					} else {
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						q.Add(param.Name, strings.Join(strs, ","))
					}
				case "deepObject":
					for k, v := range res.Map() {
						q.Add(param.Name+"["+k+"]", v.String())
					}
				}
			} else {
				q.Add(param.Name, res.String())
			}
		}
	}
	return q
}

// handlePathParameters extracts each path parameter from the input JSON and replaces its placeholder in the URL path.
func handlePathParameters(path string, params []Parameter, input string) string {
	for _, param := range params {
		res := gjson.Get(input, param.Name)
		if res.Exists() {
			// If it's an array or object, handle the serialization style
			if res.IsArray() {
				switch param.Style {
				case "simple", "": // simple is the default style for path parameters
					// simple looks the same regardless of whether explode is true
					strs := make([]string, len(res.Array()))
					for i, item := range res.Array() {
						strs[i] = item.String()
					}
					path = strings.Replace(path, "{"+param.Name+"}", strings.Join(strs, ","), 1)
				case "label":
					strs := make([]string, len(res.Array()))
					for i, item := range res.Array() {
						strs[i] = item.String()
					}

					if param.Explode == nil || !*param.Explode { // default is to not explode
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, ","), 1)
					} else {
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, "."), 1)
					}
				case "matrix":
					strs := make([]string, len(res.Array()))
					for i, item := range res.Array() {
						strs[i] = item.String()
					}

					if param.Explode == nil || !*param.Explode { // default is to not explode
						path = strings.Replace(path, "{"+param.Name+"}", ";"+param.Name+"="+strings.Join(strs, ","), 1)
					} else {
						s := ""
						for _, str := range strs {
							s += ";" + param.Name + "=" + str
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					}
				}
			} else if res.IsObject() {
				switch param.Style {
				case "simple", "":
					if param.Explode == nil || !*param.Explode { // default is to not explode
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", strings.Join(strs, ","), 1)
					} else {
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k+"="+v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", strings.Join(strs, ","), 1)
					}
				case "label":
					if param.Explode == nil || !*param.Explode { // default is to not explode
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, ","), 1)
					} else {
						s := ""
						for k, v := range res.Map() {
							s += "." + k + "=" + v.String()
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					}
				case "matrix":
					if param.Explode == nil || !*param.Explode { // default is to not explode
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", ";"+param.Name+"="+strings.Join(strs, ","), 1)
					} else {
						s := ""
						for k, v := range res.Map() {
							s += ";" + k + "=" + v.String()
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					}
				}
			} else {
				// Serialization is handled slightly differently even for basic types.
				// Explode doesn't do anything though.
				switch param.Style {
				case "simple", "":
					path = strings.Replace(path, "{"+param.Name+"}", res.String(), 1)
				case "label":
					path = strings.Replace(path, "{"+param.Name+"}", "."+res.String(), 1)
				case "matrix":
					path = strings.Replace(path, "{"+param.Name+"}", ";"+param.Name+"="+res.String(), 1)
				}
			}
		}
	}
	return path
}

// handleHeaderParameters extracts each header parameter from the input JSON and adds it to the request headers.
func handleHeaderParameters(req *http.Request, params []Parameter, input string) {
	for _, param := range params {
		res := gjson.Get(input, param.Name)
		if res.Exists() {
			if res.IsArray() {
				strs := make([]string, len(res.Array()))
				for i, item := range res.Array() {
					strs[i] = item.String()
				}
				req.Header.Add(param.Name, strings.Join(strs, ","))
			} else if res.IsObject() {
				// Handle explosion
				var strs []string
				if param.Explode == nil || !*param.Explode { // default is to not explode
					for k, v := range res.Map() {
						strs = append(strs, k, v.String())
					}
				} else {
					for k, v := range res.Map() {
						strs = append(strs, k+"="+v.String())
					}
				}
				req.Header.Add(param.Name, strings.Join(strs, ","))
			} else { // basic type
				req.Header.Add(param.Name, res.String())
			}
		}
	}
}

// handleCookieParameters extracts each cookie parameter from the input JSON and adds it to the request cookies.
func handleCookieParameters(req *http.Request, params []Parameter, input string) {
	for _, param := range params {
		res := gjson.Get(input, param.Name)
		if res.Exists() {
			if res.IsArray() {
				strs := make([]string, len(res.Array()))
				for i, item := range res.Array() {
					strs[i] = item.String()
				}
				req.AddCookie(&http.Cookie{
					Name:  param.Name,
					Value: strings.Join(strs, ","),
				})
			} else if res.IsObject() {
				var strs []string
				for k, v := range res.Map() {
					strs = append(strs, k, v.String())
				}
				req.AddCookie(&http.Cookie{
					Name:  param.Name,
					Value: strings.Join(strs, ","),
				})
			} else { // basic type
				req.AddCookie(&http.Cookie{
					Name:  param.Name,
					Value: res.String(),
				})
			}
		}
	}
}
