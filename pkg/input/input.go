package input

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/loader"
)

func FromArgs(args []string) string {
	return strings.Join(args, " ")
}

func FromCLI(file string, args []string) (string, error) {
	toolInput, err := FromFile(file)
	if err != nil || toolInput != "" {
		return toolInput, err
	}

	return FromArgs(args[1:]), nil
}

func FromFile(file string) (string, error) {
	if file == "-" {
		log.Debugf("reading stdin")
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	} else if file != "" {
		log.Debugf("reading file %s", file)
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", file, err)
		}
		return string(data), nil
	}

	return "", nil
}

// FromLocation takes a string that can be a file path or a URL to a file and returns the content of that file.
func FromLocation(s string) (string, error) {
	// Attempt to read the file first, if that fails, try to load the URL. Finally,
	// return an error if both fail.
	content, err := FromFile(s)
	if err != nil {
		log.Debugf("failed to read file %s (due to %v) attempting to load the URL...", s, err)
		content, err = loader.ContentFromURL(s)
		if err != nil {
			return "", err
		}
		// If the content is empty and there was no error, this is not a remote file. Return a generic
		// error indicating that the file could not be loaded.
		if content == "" {
			return "", fmt.Errorf("failed to load %v", s)
		}
	}
	return content, nil
}
