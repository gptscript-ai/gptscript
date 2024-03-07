package engine

import (
	"crypto/md5"
	"encoding/hex"
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

	if validToolName.MatchString(tool) {
		return tool
	}

	name := invalidChars.ReplaceAllString(tool, "-")
	if len(name) > 55 {
		name = name[:55]
	}

	hash := md5.Sum([]byte(tool))
	hexed := hex.EncodeToString(hash[:])

	return name + "-" + hexed[:8]
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
