package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/spf13/cobra"
)

type Fmt struct {
	Write bool `usage:"Write output to file instead of stdout" short:"w"`
}

func (e *Fmt) Customize(cmd *cobra.Command) {
	cmd.Args = cobra.ExactArgs(1)
}

func (e *Fmt) Run(_ *cobra.Command, args []string) error {
	input, err := input.FromFile(args[0])
	if err != nil {
		return err
	}

	var (
		doc parser.Document
		loc = locationName(args[0])
	)
	if strings.HasPrefix(input, "{") {
		if err := json.Unmarshal([]byte(input), &doc); err != nil {
			return err
		}
	} else {
		doc, err = parser.Parse(strings.NewReader(input), parser.Options{
			Location: locationName(args[0]),
		})
		if err != nil {
			return err
		}
	}

	if e.Write && loc != "" {
		return os.WriteFile(loc, []byte(doc.String()), 0644)
	}

	fmt.Print(doc.String())
	return nil
}
