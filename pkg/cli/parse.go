package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/input"
	"github.com/gptscript-ai/gptscript/pkg/loader"
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
	var (
		content string
		err     error
	)

	// Attempt to read the file first, if that fails, try to load the URL. Finally,
	// return an error if both fail.
	content, err = input.FromFile(args[0])
	if err != nil {
		log.Debugf("failed to read file %s (due to %v) attempting to load the URL...", args[0], err)
		content, err = loader.ContentFromURL(args[0])
		if err != nil {
			return err
		}
		// If the content is empty and there was no error, this is not a remote file. Return a generic
		// error indicating that the file could not be loaded.
		if content == "" {
			return fmt.Errorf("failed to load %v", args[0])
		}
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
