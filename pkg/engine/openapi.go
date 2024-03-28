package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/tidwall/gjson"
)

var (
	SupportedMIMETypes       = []string{"application/json", "text/plain", "multipart/form-data"}
	SupportedSecuritySchemes = []string{"oauth2"}
)

type Parameter struct {
	Name    string `json:"name"`
	Style   string `json:"style"`
	Explode *bool  `json:"explode"`
}

type OAuthInfo struct {
	AuthorizationURL string   `json:"authorizationURL"`
	TokenURL         string   `json:"tokenURL"`
	Flow             string   `json:"flow"`
	Scopes           []string `json:"scopes"`
}

type OpenAPIInstructions struct {
	Server           string      `json:"server"`
	Path             string      `json:"path"`
	Method           string      `json:"method"`
	BodyContentMIME  string      `json:"bodyContentMIME"`
	QueryParameters  []Parameter `json:"queryParameters"`
	PathParameters   []Parameter `json:"pathParameters"`
	HeaderParameters []Parameter `json:"headerParameters"`
	CookieParameters []Parameter `json:"cookieParameters"`
}

func (e *Engine) runOpenAPI(ctx context.Context, prg *types.Program, tool types.Tool, input string) (*Return, error) {
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
	u, err := url.Parse(instructions.Server + instructions.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %s: %w", instructions.Server+instructions.Path, err)
	}

	// Set up the request
	req, err := http.NewRequest(instructions.Method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Check for a bearer token (only if using HTTPS)
	if u.Scheme == "https" {
		// For "https://example.com" the bearer token env name would be GPTSCRIPT_EXAMPLE_COM_BEARER_TOKEN
		bearerEnv := "GPTSCRIPT_" + strings.ToUpper(strings.Replace(u.Host, ".", "_", -1)) + "_BEARER_TOKEN"
		if bearerToken, ok := envMap[bearerEnv]; ok {
			req.Header.Set("Authorization", "Bearer "+bearerToken)
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
		if res.Exists() {
			var body bytes.Buffer
			switch instructions.BodyContentMIME {
			case "application/json":
				if err := json.NewEncoder(&body).Encode(res.Value()); err != nil {
					return nil, fmt.Errorf("failed to encode JSON: %w", err)
				}
				req.Header.Set("Content-Type", "application/json")

			case "text/plain":
				body.WriteString(res.String())
				req.Header.Set("Content-Type", "text/plain")

			case "multipart/form-data":
				multiPartWriter := multipart.NewWriter(&body)
				req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())
				if res.IsObject() {
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

					if param.Explode != nil && *param.Explode {
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, "."), 1)
					} else {
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, ","), 1)
					}
				case "matrix":
					strs := make([]string, len(res.Array()))
					for i, item := range res.Array() {
						strs[i] = item.String()
					}

					if param.Explode != nil && *param.Explode {
						s := ""
						for _, str := range strs {
							s += ";" + param.Name + "=" + str
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					} else {
						path = strings.Replace(path, "{"+param.Name+"}", ";"+param.Name+"="+strings.Join(strs, ","), 1)
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
					if param.Explode != nil && *param.Explode {
						s := ""
						for k, v := range res.Map() {
							s += "." + k + "=" + v.String()
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					} else {
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", "."+strings.Join(strs, ","), 1)
					}
				case "matrix":
					if param.Explode != nil && *param.Explode {
						s := ""
						for k, v := range res.Map() {
							s += ";" + k + "=" + v.String()
						}
						path = strings.Replace(path, "{"+param.Name+"}", s, 1)
					} else {
						var strs []string
						for k, v := range res.Map() {
							strs = append(strs, k, v.String())
						}
						path = strings.Replace(path, "{"+param.Name+"}", ";"+param.Name+"="+strings.Join(strs, ","), 1)
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
