package loader

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
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

	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

const (
	GithubPrefix = "github.com/"
	githubRawURL = "https://raw.githubusercontent.com/"
)

type source struct {
	Content io.ReadCloser
	Remote  bool
	Path    string
	Name    string
	File    string
}

func (s *source) String() string {
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

func loadLocal(base *source, name string) (*source, bool, error) {
	path := filepath.Join(base.Path, name)

	content, ok, err := openFile(path)
	if err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	log.Debugf("opened %s", path)

	return &source{
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

func loadURL(ctx context.Context, base *source, name string) (*source, bool, error) {
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

	return &source{
		Content: resp.Body,
		Remote:  true,
		Path:    pathURL.String(),
		Name:    filepath.Base(parsed.Path),
		File:    url,
	}, true, nil
}

func loadProgram(data []byte, into *types.Program, targetToolName string) (types.Tool, error) {
	var (
		ext types.Program
		id  string
	)

	summed := sha256.Sum256(data)
	id = "@" + hex.EncodeToString(summed[:])[:12]

	if err := json.Unmarshal(data[len(assemble.Header):], &ext); err != nil {
		return types.Tool{}, err
	}

	for k, v := range ext.ToolSet {
		for tk, tv := range v.ToolMapping {
			v.ToolMapping[tk] = tv + id
		}
		v.ID = k + id
		into.ToolSet[v.ID] = v
	}

	tool := into.ToolSet[ext.EntryToolID+id]
	if targetToolName == "" {
		return tool, nil
	}

	tool, ok := into.ToolSet[tool.LocalTools[targetToolName]]
	if !ok {
		return tool, &engine.ErrToolNotFound{
			ToolName: targetToolName,
		}
	}

	return tool, nil
}

func readTool(ctx context.Context, prg *types.Program, base *source, targetToolName string) (types.Tool, error) {
	data, err := io.ReadAll(base.Content)
	if err != nil {
		return types.Tool{}, err
	}
	_ = base.Content.Close()

	if bytes.HasPrefix(data, assemble.Header) {
		return loadProgram(data, prg, targetToolName)
	}

	tools, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return types.Tool{}, err
	}

	if len(tools) == 0 {
		return types.Tool{}, fmt.Errorf("no tools found in %s", base)
	}

	var (
		localTools = types.ToolSet{}
		mainTool   types.Tool
	)

	for i, tool := range tools {
		tool.Source.File = base.File

		// Probably a better way to come up with an ID
		tool.ID = tool.Source.String()

		if i == 0 {
			mainTool = tool
		}

		if i != 0 && tool.Name == "" {
			return types.Tool{}, parser.NewErrLine(tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have no name"))
		}

		if targetToolName != "" && tool.Name == targetToolName {
			mainTool = tool
		}

		if existing, ok := localTools[tool.Name]; ok {
			return types.Tool{}, parser.NewErrLine(tool.Source.LineNo,
				fmt.Errorf("duplicate tool name [%s] in %s found at lines %d and %d", tool.Name, tool.Source.File,
					tool.Source.LineNo, existing.Source.LineNo))
		}

		localTools[tool.Name] = tool
	}

	return link(ctx, prg, base, mainTool, localTools)
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

func link(ctx context.Context, prg *types.Program, base *source, tool types.Tool, localTools types.ToolSet) (types.Tool, error) {
	if existing, ok := prg.ToolSet[tool.ID]; ok {
		return existing, nil
	}

	tool.ToolMapping = map[string]string{}
	tool.LocalTools = map[string]string{}
	toolNames := map[string]struct{}{}

	// Add now to break circular loops, but later we will update this tool and copy the new
	// tool to the prg.ToolSet
	prg.ToolSet[tool.ID] = tool

	// The below is done in two loops so that local names stay as the tool names
	// and don't get mangled by external references

	for _, targetToolName := range tool.Tools {
		localTool, ok := localTools[targetToolName]
		if !ok {
			continue
		}

		var linkedTool types.Tool
		if existing, ok := prg.ToolSet[localTool.ID]; ok {
			linkedTool = existing
		} else {
			var err error
			linkedTool, err = link(ctx, prg, base, localTool, localTools)
			if err != nil {
				return types.Tool{}, fmt.Errorf("failed linking %s at %s: %w", targetToolName, base, err)
			}
		}

		tool.ToolMapping[targetToolName] = linkedTool.ID
		toolNames[targetToolName] = struct{}{}
	}

	for i, targetToolName := range tool.Tools {
		_, ok := localTools[targetToolName]
		if ok {
			continue
		}

		toolName, subTool, ok := strings.Cut(targetToolName, " from ")
		if ok {
			toolName = strings.TrimSpace(toolName)
			subTool = strings.TrimSpace(subTool)
		}

		resolvedTool, err := resolve(ctx, prg, base, toolName, subTool)
		if err != nil {
			return types.Tool{}, fmt.Errorf("failed resolving %s at %s: %w", targetToolName, base, err)
		}

		newToolName := pickToolName(toolName, toolNames)
		tool.ToolMapping[newToolName] = resolvedTool.ID
		tool.Tools[i] = newToolName
	}

	for _, localTool := range localTools {
		tool.LocalTools[localTool.Name] = localTool.ID
	}

	tool = builtin.SetDefaults(tool)
	prg.ToolSet[tool.ID] = tool

	return tool, nil
}

func Program(ctx context.Context, name, subToolName string) (types.Program, error) {
	prg := types.Program{
		ToolSet: types.ToolSet{},
	}
	tool, err := resolve(ctx, &prg, &source{}, name, subToolName)
	if err != nil {
		return types.Program{}, err
	}
	prg.EntryToolID = tool.ID
	return prg, nil
}

func resolve(ctx context.Context, prg *types.Program, base *source, name, subTool string) (types.Tool, error) {
	if subTool == "" {
		t, ok := builtin.Builtin(name)
		if ok {
			prg.ToolSet[t.ID] = t
			return t, nil
		}
	}

	s, err := input(ctx, base, name)
	if err != nil {
		return types.Tool{}, err
	}

	return readTool(ctx, prg, s, subTool)
}

func input(ctx context.Context, base *source, name string) (*source, error) {
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
