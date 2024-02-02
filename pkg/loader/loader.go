package loader

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	url2 "net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/acorn-io/gptscript/pkg/assemble"
	"github.com/acorn-io/gptscript/pkg/parser"
	"github.com/acorn-io/gptscript/pkg/types"
)

const (
	GithubPrefix = "github.com/"
	githubRawURL = "https://raw.githubusercontent.com/"
)

type Source struct {
	Content io.ReadCloser
	Remote  bool
	Path    string
	Name    string
	File    string
}

func (s *Source) String() string {
	if s.Path == "" && s.Name == "" {
		return ""
	}
	return s.Path + "/" + s.Name
}

func openFile(path string) (io.ReadCloser, bool, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return f, true, nil
}

func loadLocal(base *Source, name string) (*Source, bool, error) {
	path := filepath.Join(base.Path, name)

	content, ok, err := openFile(path)
	if err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	log.Debugf("opened %s", path)

	return &Source{
		Content: content,
		Remote:  false,
		Path:    filepath.Dir(path),
		Name:    filepath.Base(path),
		File:    path,
	}, true, nil
}

func githubURL(urlName string) (string, bool) {
	if !strings.HasPrefix(urlName, GithubPrefix) {
		return "", false
	}

	url, version, _ := strings.Cut(urlName, "@")
	if version == "" {
		version = "HEAD"
	}

	parts := strings.Split(url, "/")
	// Must be at least 4 parts github.com/ACCOUNT/REPO/FILE
	if len(parts) < 4 {
		return "", false
	}

	url = githubRawURL + parts[1] + "/" + parts[2] + "/" + version + "/" + strings.Join(parts[3:], "/")
	return url, true
}

func loadURL(ctx context.Context, base *Source, name string) (*Source, bool, error) {
	url := name
	if base.Path != "" {
		url = base.Path + "/" + name
	}
	if githubURL, ok := githubURL(url); ok {
		url = githubURL
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("error loading %s: %s", url, resp.Status)
	}

	log.Debugf("opened %s", url)

	parsed, err := url2.Parse(url)
	if err != nil {
		return nil, false, err
	}

	pathURL := *parsed
	pathURL.Path = filepath.Dir(parsed.Path)

	return &Source{
		Content: resp.Body,
		Remote:  true,
		Path:    pathURL.String(),
		Name:    filepath.Base(parsed.Path),
		File:    url,
	}, true, nil
}

func ReadTool(ctx context.Context, base *Source, targetToolName string) (*types.Tool, error) {
	data, err := io.ReadAll(base.Content)
	if err != nil {
		return nil, err
	}
	_ = base.Content.Close()

	if bytes.HasPrefix(data, assemble.Header) {
		var tool types.Tool
		return &tool, json.Unmarshal(data[len(assemble.Header):], &tool)
	}

	tools, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools found in %s", base)
	}

	var (
		toolSet  = types.ToolSet{}
		mainTool types.Tool
	)

	for i, tool := range tools {
		tool.Source.File = base.File
		if i == 0 {
			mainTool = tool
		}

		if i != 0 && tool.Name == "" {
			return nil, parser.NewErrLine(tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have no name"))
		}

		if targetToolName != "" && tool.Name == targetToolName {
			mainTool = tool
		}

		toolSet[tool.Name] = tool
	}

	return link(ctx, base, mainTool, toolSet)
}

var (
	validToolName = regexp.MustCompile("^[a-zA-Z0-9_-]{1,64}$")
	invalidChars  = regexp.MustCompile("[^a-zA-Z0-9_-]+")
)

func toolNormalizer(tool string) string {
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

func pickToolName(toolName string, existing map[string]struct{}) string {
	newName, suffix, ok := strings.Cut(toolName, "/")
	if ok {
		newName = suffix
	}
	newName = strings.TrimSuffix(newName, filepath.Ext(newName))
	if newName == "" {
		newName = "external"
	}

	for {
		testName := toolNormalizer(newName)
		if _, ok := existing[testName]; !ok {
			existing[testName] = struct{}{}
			return testName
		}
		newName += "0"
	}
}

func link(ctx context.Context, base *Source, tool types.Tool, toolSet types.ToolSet) (*types.Tool, error) {
	tool.ToolSet = types.ToolSet{}
	toolNames := map[string]struct{}{}

	for _, targetToolName := range tool.Tools {
		targetTool, ok := toolSet[targetToolName]
		if !ok {
			continue
		}

		linkedTool, err := link(ctx, base, targetTool, toolSet)
		if err != nil {
			return nil, fmt.Errorf("failed linking %s at %s: %w", targetToolName, base, err)
		}
		tool.ToolSet[targetToolName] = *linkedTool
		toolNames[targetToolName] = struct{}{}
	}

	for i, targetToolName := range tool.Tools {
		_, ok := toolSet[targetToolName]
		if ok {
			continue
		}

		toolName, subTool, ok := strings.Cut(targetToolName, " from ")
		if ok {
			toolName = strings.TrimSpace(toolName)
			subTool = strings.TrimSpace(subTool)
		}
		resolvedTool, err := Resolve(ctx, base, toolName, subTool)
		if err != nil {
			return nil, fmt.Errorf("failed resolving %s at %s: %w", targetToolName, base, err)
		}
		newToolName := pickToolName(toolName, toolNames)
		tool.ToolSet[newToolName] = *resolvedTool
		tool.Tools[i] = newToolName
	}

	return &tool, nil
}

func Tool(ctx context.Context, name, subToolName string) (*types.Tool, error) {
	return Resolve(ctx, &Source{}, name, subToolName)
}

func Resolve(ctx context.Context, base *Source, name, subTool string) (*types.Tool, error) {
	s, err := Input(ctx, base, name)
	if err != nil {
		return nil, err
	}

	return ReadTool(ctx, s, subTool)
}

func Input(ctx context.Context, base *Source, name string) (*Source, error) {
	if !base.Remote {
		s, ok, err := loadLocal(base, name)
		if err != nil || ok {
			return s, err
		}
	}

	s, ok, err := loadURL(ctx, base, name)
	if err != nil || ok {
		return s, err
	}

	return nil, fmt.Errorf("can not load tools path=%s name=%s", base.Path, name)
}
