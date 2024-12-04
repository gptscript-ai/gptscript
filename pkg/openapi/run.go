package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/exp/maps"
)

const RunTool = "run"

func Run(operationID, defaultHost, args string, t *openapi3.T, envs []string) (string, bool, error) {
	envMap := make(map[string]string, len(envs))
	for _, e := range envs {
		k, v, _ := strings.Cut(e, "=")
		envMap[k] = v
	}

	if args == "" {
		args = "{}"
	}
	schemaJSON, opInfo, found, err := GetSchema(operationID, defaultHost, t)
	if err != nil || !found {
		return "", false, err
	}

	// Validate args against the schema.
	validationResult, err := gojsonschema.Validate(gojsonschema.NewStringLoader(schemaJSON), gojsonschema.NewStringLoader(args))
	if err != nil {
		// We don't return an error here because we want the LLM to be able to maintain control and try again.
		return fmt.Sprintf("ERROR: failed to validate arguments. Make sure your arguments are valid JSON. %v", err), true, nil
	}

	if !validationResult.Valid() {
		// We don't return an error here because we want the LLM to be able to maintain control and try again.
		return fmt.Sprintf("invalid arguments for operation %s: %s", operationID, validationResult.Errors()), true, nil
	}

	// Construct and execute the HTTP request.

	// Handle path parameters.
	opInfo.Path = HandlePathParameters(opInfo.Path, opInfo.PathParams, args)

	// Parse the URL
	path, err := url.JoinPath(opInfo.Server, opInfo.Path)
	if err != nil {
		return "", false, fmt.Errorf("failed to join server and path: %w", err)
	}

	u, err := url.Parse(path)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse server URL %s: %w", opInfo.Server+opInfo.Path, err)
	}

	// Set up the request
	req, err := http.NewRequest(opInfo.Method, u.String(), nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to create request: %w", err)
	}

	// Check for authentication
	if len(opInfo.SecurityInfos) > 0 {
		if err := HandleAuths(req, envMap, opInfo.SecurityInfos); err != nil {
			return "", false, fmt.Errorf("error setting up authentication: %w", err)
		}
	}

	// If there is a bearer token set for the whole server, and no Authorization header has been defined, use it.
	if token, ok := envMap["GPTSCRIPT_"+env.ToEnvLike(u.Hostname())+"_BEARER_TOKEN"]; ok {
		if req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	// Handle query parameters
	req.URL.RawQuery = HandleQueryParameters(req.URL.Query(), opInfo.QueryParams, args).Encode()

	// Handle header and cookie parameters
	HandleHeaderParameters(req, opInfo.HeaderParams, args)
	HandleCookieParameters(req, opInfo.CookieParams, args)

	// Handle request body
	if opInfo.BodyContentMIME != "" {
		res := gjson.Get(args, "requestBodyContent")
		var body bytes.Buffer
		switch opInfo.BodyContentMIME {
		case "application/json":
			var reqBody any = struct{}{}
			if res.Exists() {
				reqBody = res.Value()
			}
			if err := json.NewEncoder(&body).Encode(reqBody); err != nil {
				return "", false, fmt.Errorf("failed to encode JSON: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.ContentLength = int64(body.Len())

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
						return "", false, fmt.Errorf("failed to write multipart field: %w", err)
					}
				}
			} else {
				return "", false, fmt.Errorf("multipart/form-data requires an object as the requestBodyContent")
			}
			if err := multiPartWriter.Close(); err != nil {
				return "", false, fmt.Errorf("failed to close multipart writer: %w", err)
			}

		default:
			return "", false, fmt.Errorf("unsupported MIME type: %s", opInfo.BodyContentMIME)
		}
		req.Body = io.NopCloser(&body)
	}

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("failed to read response: %w", err)
	}

	return string(result), true, nil
}

// HandleAuths will set up the request with the necessary authentication information.
// A set of sets of SecurityInfo is passed in, where each represents a possible set of security options.
func HandleAuths(req *http.Request, envMap map[string]string, infoSets [][]SecurityInfo) error {
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
		v := url.Values{}
		for _, info := range infoSet {
			envNames := maps.Values(info.getCredentialNamesAndEnvVars(req.URL.Hostname()))
			switch info.Type {
			case "apiKey":
				switch info.In {
				case "header":
					req.Header.Set(info.APIKeyName, envMap[envNames[0]])
				case "query":
					v.Add(info.APIKeyName, envMap[envNames[0]])
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
		if len(v) > 0 {
			req.URL.RawQuery = v.Encode()
		}
		return nil
	}

	return fmt.Errorf("did not find the needed environment variables for any of the security options. "+
		"At least one of these sets of environment variables must be provided: %v", missingVariables)
}

// HandlePathParameters extracts each path parameter from the input JSON and replaces its placeholder in the URL path.
func HandlePathParameters(path string, params []Parameter, input string) string {
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

// HandleQueryParameters extracts each query parameter from the input JSON and adds it to the URL query.
func HandleQueryParameters(q url.Values, params []Parameter, input string) url.Values {
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

// HandleHeaderParameters extracts each header parameter from the input JSON and adds it to the request headers.
func HandleHeaderParameters(req *http.Request, params []Parameter, input string) {
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

// HandleCookieParameters extracts each cookie parameter from the input JSON and adds it to the request cookies.
func HandleCookieParameters(req *http.Request, params []Parameter, input string) {
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
