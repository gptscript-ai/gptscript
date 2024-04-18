package types

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/system"
)

var (
	validToolName = regexp.MustCompile("^[a-zA-Z0-9]{1,64}$")
	invalidChars  = regexp.MustCompile("[^a-zA-Z0-9_]+")
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

	if len(tool) > 55 {
		tool = tool[:55]
	}

	tool = invalidChars.ReplaceAllString(tool, "_")

	var result []string
	for i, part := range strings.Split(tool, "_") {
		lower := strings.ToLower(part)
		if i != 0 && len(lower) > 0 {
			lower = strings.ToTitle(lower[0:1]) + lower[1:]
		}
		result = append(result, lower)
	}

	return strings.Join(result, "")
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
