package vision

import (
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	Schema = types.JSONSchema{
		Property: types.Property{
			Type: "object",
		},
		Properties: map[string]types.Property{
			"base64": {
				Description: "The base64 encoded value of the image if an image URL is not specified",
				Type:        "string",
			},
			"contentType": {
				Description: `The content type of the image such as "image/jpeg" or "image/png"`,
				Type:        "string",
			},
			"text": {
				Description: "Instructions on how the passed image should be analyzed",
				Type:        "string",
			},
			"url": {
				Description: "The URL to the image to be processed. This should be set if base64 is not set",
				Type:        "string",
			},
		},
		Defs: map[string]types.JSONSchema{},
	}
)

type inputMessage struct {
	Text        string `json:"text,omitempty"`
	Base64      string `json:"base64,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	URL         string `json:"url,omitempty"`
}
