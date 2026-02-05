package printer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var (
	prettyPrint   = os.Getenv("NANOBOT_LOG_PRETTY_PRINT") != ""
	noColors      = os.Getenv("NANOBOT_NO_COLORS") != "" || os.Getenv("NO_COLOR") != ""
	printLock     sync.Mutex
	lastPrefix    string
	currentLine   string
	longestPrefix int

	termColorToAscii = map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
	}
	lightVariants = map[string]string{
		"red":     "\033[91m",
		"green":   "\033[92m",
		"yellow":  "\033[93m",
		"blue":    "\033[94m",
		"magenta": "\033[95m",
	}
	colors = []string{
		"green", "yellow", "blue", "magenta", "red",
	}
	lastColorIndex = 0
	prefixToColor  = map[string]string{}
)

func appendToLine(prefix, content string) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		if remaining := width - len(currentLine); len(content) > remaining {
			appendToLine(prefix, content[:remaining])
			newline()
			printPrefix(prefix, content[remaining:])
			return
		}
	}

	_, _ = fmt.Fprint(os.Stderr, content)
	currentLine += content
}

func newline() {
	_, _ = fmt.Fprint(os.Stderr, "\n")
	currentLine = ""
	lastPrefix = ""
}

func printPrefix(prefix, content string) {
	if lastPrefix == "" {
		appendToLine(prefix, prefix+" "+content)
	} else if lastPrefix == prefix {
		appendToLine(prefix, content)
	} else if lastPrefix != prefix {
		newline()
		appendToLine(prefix, prefix+" "+content)
	}
	lastPrefix = prefix
}

func formatPrefix(prefix string) string {
	if len(prefix) < 3 || noColors {
		return prefix
	}

	key := strings.ReplaceAll(prefix[2:], " ", "")

	colorsCodes := termColorToAscii
	if strings.HasPrefix(prefix, "-") {
		colorsCodes = lightVariants
	}

	if color, ok := prefixToColor[key]; ok {
		return color + prefix + "\033[0m"
	}

	if lastColorIndex >= len(colors) {
		lastColorIndex = 0
	}

	color := colorsCodes[colors[lastColorIndex]]
	prefixToColor[key] = color
	lastColorIndex++

	return color + prefix + "\033[0m"
}

func Prefix(prefix, content string) {
	if content == "" {
		return
	}

	if prettyPrint {
		jsonData := map[string]any{}
		if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
			if jsonFormatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				content = string(jsonFormatted)
			}
		}
	}

	printLock.Lock()
	defer printLock.Unlock()

	if len(prefix) > longestPrefix {
		longestPrefix = len(prefix)
	}

	// pad prefix to longestPrefix
	if len(prefix) < longestPrefix {
		prefix += strings.Repeat(" ", longestPrefix-len(prefix))
	}

	prefix = formatPrefix(prefix + "│")

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i > 0 {
			newline()

			if i == len(lines)-1 && line == "" {
				continue // Skip empty lines at the end
			}
		}

		printPrefix(prefix, line)
	}
}
