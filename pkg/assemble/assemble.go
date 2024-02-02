package assemble

import (
	"context"
	"encoding/json"
	"io"

	"github.com/acorn-io/gptscript/pkg/types"
)

var Header = []byte("GPTSCRIPT\x00")

func Assemble(ctx context.Context, tool types.Tool, output io.Writer) error {
	if _, err := output.Write(Header); err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(tool)
}
