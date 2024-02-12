package vision

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	urlBase = os.Getenv("cached://")
)

func ToVisionMessage(c *cache.Client, message types.CompletionMessage) (types.CompletionMessage, error) {
	if len(message.Content) != 1 || !strings.HasPrefix(message.Content[0].Text, "{") {
		return message, nil
	}

	var (
		input   inputMessage
		content = message.Content[0]
	)
	if err := json.Unmarshal([]byte(content.Text), &input); err != nil {
		return message, nil
	}

	content.Text = input.Text

	if input.URL != "" {
		b64, ok, err := Base64FromStored(c, input.URL)
		if err != nil {
			return message, err
		}
		if b64 == "" || !ok {
			content.Image = &types.ImageURL{
				URL: input.URL,
			}
		} else {
			input.Base64 = b64
		}
	}

	if input.Base64 != "" && input.ContentType != "" {
		content.Image = &types.ImageURL{
			Base64:      input.Base64,
			ContentType: input.ContentType,
		}
	}

	message.Content = []types.ContentPart{
		content,
	}

	return message, nil
}

func Base64FromStored(cache *cache.Client, url string) (string, bool, error) {
	if !strings.HasPrefix(url, urlBase) {
		return "", false, nil
	}
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", false, nil
	}
	name := parts[len(parts)-1]

	cached, ok, err := cache.Get(name)
	if err != nil || !ok {
		return "", ok, err
	}

	return base64.StdEncoding.EncodeToString(cached), true, nil
}

func ImageToURL(c *cache.Client, vision bool, message types.ImageURL) (string, error) {
	if message.URL != "" {
		return message.URL, nil
	}

	if vision {
		return fmt.Sprintf("data:%s;base64,%s", message.ContentType, message.Base64), nil
	}

	data, err := base64.StdEncoding.DecodeString(message.Base64)
	if err != nil {
		return "", err
	}

	id := "i" + hash.Encode(message)[:12]
	if err := c.Store(id, data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", urlBase, id), nil
}
