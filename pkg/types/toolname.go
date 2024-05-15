package types

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/system"
)

var (
	validToolName = regexp.MustCompile("^[a-zA-Z0-9]{1,64}$")
	invalidChars  = regexp.MustCompile("[^a-zA-Z0-9_]+")
)

func ToolNormalizer(tool string) string {
	_, subTool := SplitToolRef(tool)
	lastTool := tool
	if subTool != "" {
		lastTool = subTool
	}

	parts := strings.Split(lastTool, "/")
	tool = parts[len(parts)-1]
	if parts[len(parts)-1] == "tool.gpt" && len(parts) > 1 && len(parts[len(parts)-2]) > 2 {
		tool = parts[len(parts)-2]
	}
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

	var (
		result   []string
		appended bool
	)
	for _, part := range strings.Split(tool, "_") {
		lower := strings.ToLower(part)
		if appended && len(lower) > 0 {
			lower = strings.ToTitle(lower[0:1]) + lower[1:]
		}
		if lower != "" {
			result = append(result, lower)
			appended = true
		}
	}

	final := strings.Join(result, "")
	if final == "" {
		return "tool"
	}
	return final
}

func SplitToolRef(targetToolName string) (toolName, subTool string) {
	var (
		fields = strings.Fields(targetToolName)
		idx    = slices.Index(fields, "from")
	)

	defer func() {
		toolName, _ = SplitArg(toolName)
	}()

	if idx == -1 {
		return strings.TrimSpace(targetToolName), ""
	}

	return strings.Join(fields[idx+1:], " "),
		strings.Join(fields[:idx], " ")
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
