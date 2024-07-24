package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"
)

func Load(source string) (*openapi3.T, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return loadFromURL(source)
	}
	return loadFromFile(source)
}

func loadFromURL(source string) (*openapi3.T, error) {
	resp, err := http.DefaultClient.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	contents, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return loadFromBytes(contents)
}

func loadFromFile(source string) (*openapi3.T, error) {
	contents, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	return loadFromBytes(contents)
}

func loadFromBytes(content []byte) (*openapi3.T, error) {
	var (
		openAPIDocument *openapi3.T
		err             error
	)

	switch IsOpenAPI(content) {
	case 2:
		// Convert OpenAPI v2 to v3
		jsonContent := content
		if !json.Valid(content) {
			jsonContent, err = kyaml.YAMLToJSON(content)
			if err != nil {
				return nil, err
			}
		}

		doc := &openapi2.T{}
		if err := doc.UnmarshalJSON(jsonContent); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OpenAPI v2 document: %w", err)
		}

		openAPIDocument, err = openapi2conv.ToV3(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert OpenAPI v2 to v3: %w", err)
		}
	case 3:
		openAPIDocument, err = openapi3.NewLoader().LoadFromData(content)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported OpenAPI version")
	}

	return openAPIDocument, nil
}

// IsOpenAPI checks if the data is an OpenAPI definition and returns the version if it is.
func IsOpenAPI(data []byte) int {
	var fragment struct {
		Paths   map[string]any `json:"paths,omitempty"`
		Swagger string         `json:"swagger,omitempty"`
		OpenAPI string         `json:"openapi,omitempty"`
	}

	if err := json.Unmarshal(data, &fragment); err != nil {
		if err := yaml.Unmarshal(data, &fragment); err != nil {
			return 0
		}
	}
	if len(fragment.Paths) == 0 {
		return 0
	}

	if v, _, _ := strings.Cut(fragment.OpenAPI, "."); v != "" {
		ver, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return ver
	}

	if v, _, _ := strings.Cut(fragment.Swagger, "."); v != "" {
		ver, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return ver
	}

	return 0
}
