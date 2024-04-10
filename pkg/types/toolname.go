package types

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/system"
)

var (
	validToolName = regexp.MustCompile("^[a-zA-Z0-9_-]{1,64}$")
	invalidChars  = regexp.MustCompile("[^a-zA-Z0-9_-]+")
)

func ToolNormalizer(tool string) string {
	parts := strings.Split(tool, "/")
	tool = parts[len(parts)-1]
	if strings.HasSuffix(tool, system.Suffix) {
		tool = strings.TrimSuffix(tool, filepath.Ext(tool))
	}
	tool = strings.TrimPrefix(tool, "sys.")

	if validToolName.MatchString(tool) {
		return tool
	}

	name := invalidChars.ReplaceAllString(tool, "_")
	for strings.HasSuffix(name, "_") {
		name = strings.TrimSuffix(name, "_")
	}

	if len(name) > 55 {
		name = name[:55]
	}

	return name
}

func PickToolName(toolName string, existing map[string]struct{}) string {
	if toolName == "" {
		toolName = "external"
	}

	for {
		testName := ToolNormalizer(toolName)
		if _, ok := existing[testName]; !ok {
			existing[testName] = struct{}{}
			return testName
		}
		toolName += "0"
	}
}
