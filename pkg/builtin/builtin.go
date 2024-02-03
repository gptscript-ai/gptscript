package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/acorn-io/gptscript/pkg/types"
)

var Tools = map[string]types.Tool{
	"sys.read": {
		Description: "Reads the contents of a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to read"),
		BuiltinFunc: func(ctx context.Context, env []string, input string) (string, error) {
			var params struct {
				Filename string `json:"filename,omitempty"`
			}
			if err := json.Unmarshal([]byte(input), &params); err != nil {
				return "", err
			}

			log.Debugf("Reading file %s", params.Filename)
			data, err := os.ReadFile(params.Filename)
			if err != nil {
				return "", err
			}

			if utf8.Valid(data) {
				return string(data), nil
			}
			return base64.StdEncoding.EncodeToString(data), nil
		},
	},
	"sys.write": {
		Description: "Write the contents to a file",
		Arguments: types.ObjectSchema(
			"filename", "The name of the file to write to",
			"content", "The content to write"),
		BuiltinFunc: func(ctx context.Context, env []string, input string) (string, error) {
			var params struct {
				Filename string `json:"filename,omitempty"`
				Content  string `json:"content,omitempty"`
			}
			if err := json.Unmarshal([]byte(input), &params); err != nil {
				return "", err
			}

			data := []byte(params.Content)
			msg := fmt.Sprintf("Wrote %d bytes to file %s", len(data), params.Filename)
			log.Debugf(msg)

			return "", os.WriteFile(params.Filename, data, 0644)
		},
	},
}

func Builtin(name string) (types.Tool, bool) {
	t, ok := Tools[name]
	t.Name = name
	t.ID = name
	t.Instructions = "#!" + name
	return t, ok
}
