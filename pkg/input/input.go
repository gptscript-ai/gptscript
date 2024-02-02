package input

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func FromArgs(args []string) string {
	return strings.Join(args, " ")
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
