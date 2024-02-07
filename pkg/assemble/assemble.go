package assemble

import (
	"context"
	"encoding/json"
	"io"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

var Header = []byte("GPTSCRIPT!")

func Assemble(ctx context.Context, prg types.Program, output io.Writer) error {
	if _, err := output.Write(Header); err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(prg)
}
