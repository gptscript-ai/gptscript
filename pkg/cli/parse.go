package cli

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/spf13/cobra"
)

type Parse struct {
	PrettyPrint bool `usage:"Indent the json output" short:"p"`
}

func (e *Parse) Customize(cmd *cobra.Command) {
	cmd.Args = cobra.ExactArgs(1)
}

func locationName(l string) string {
	if l == "-" {
		return ""
	}
	return l
}

func (e *Parse) Run(_ *cobra.Command, args []string) error {
	content, err := input.FromLocation(args[0])
	if err != nil {
		return err
	}

	docs, err := parser.Parse(strings.NewReader(content), parser.Options{
		Location: locationName(args[0]),
	})
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	if e.PrettyPrint {
		enc.SetIndent("", "  ")
	}

	return enc.Encode(docs)
}
