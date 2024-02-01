package loader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/acorn-io/gptscript/pkg/parser"
	"github.com/acorn-io/gptscript/pkg/types"
)

const (
	Suffix       = ".gpt"
	GithubPrefix = "github.com/"
	githubRawURL = "https://raw.githubusercontent.com/"
)

type Source struct {
	Content io.ReadCloser
	Remote  bool
	Root    string
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
	f, err := os.Open(path + Suffix)
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
		path = filepath.Join(base.Root, "vendor", path)
		content, ok, err = openFile(path)
		if err != nil {
			return nil, false, err
		} else if !ok {
			return nil, false, nil
		}
	}
	log.Debugf("opened %s%s for %s", path, Suffix, path)

	return &Source{
		Content: content,
		Remote:  false,
		Root:    base.Root,
		Path:    filepath.Dir(path),
		Name:    filepath.Base(path),
		File:    name + Suffix,
	}, true, nil
}

func loadGithub(ctx context.Context, base *Source, name string) (*Source, bool, error) {
	urlName := filepath.Join(base.Path, name)
	if !strings.HasPrefix(urlName, GithubPrefix) {
		return nil, false, nil
	}

	url, version, _ := strings.Cut(urlName, "@")
	if version == "" {
		version = "HEAD"
	}

	parts := strings.Split(url, "/")
	// Must be at least 4 parts github.com/ACCOUNT/REPO/FILE
	if len(parts) < 4 {
		return nil, false, fmt.Errorf("invalid github URL, must be at least 4 parts github.com/ACCOUNT/REPO/FILE: %s", url)
	}

	url = githubRawURL + parts[1] + "/" + parts[2] + "/" + version + "/" + strings.Join(parts[3:], "/") + Suffix
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

	log.Debugf("opened %s for %s", url, urlName)

	return &Source{
		Content: resp.Body,
		Remote:  true,
		Path:    filepath.Dir(urlName),
		Name:    filepath.Base(urlName),
		File:    url,
	}, true, nil
}

func lookupRoot(path string) (string, error) {
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		if s, err := os.Stat(filepath.Join(parent, "vendor")); err == nil && s.IsDir() {
			return parent, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		current = parent
	}

	return filepath.Dir(path), nil
}

func ReadTool(ctx context.Context, base *Source, targetToolName string) (*types.Tool, error) {
	defer base.Content.Close()

	tools, err := parser.Parse(base.Content)
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

func link(ctx context.Context, base *Source, tool types.Tool, toolSet types.ToolSet) (*types.Tool, error) {
	tool.ToolSet = types.ToolSet{}
	for _, targetToolName := range tool.Tools {
		targetTool, ok := toolSet[targetToolName]
		if ok {
			linkedTool, err := link(ctx, base, targetTool, toolSet)
			if err != nil {
				return nil, fmt.Errorf("failed linking %s at %s: %w", targetToolName, base, err)
			}
			tool.ToolSet[targetToolName] = *linkedTool
		} else {
			resolvedTool, err := Resolve(ctx, base, targetToolName, "")
			if err != nil {
				return nil, fmt.Errorf("failed resolving %s at %s: %w", targetToolName, base, err)
			}
			tool.ToolSet[targetToolName] = *resolvedTool
		}
	}

	return &tool, nil
}

func Tool(ctx context.Context, name, subToolName string) (*types.Tool, error) {
	var base Source

	name = strings.TrimSuffix(name, Suffix)

	f, ok, err := openFile(name)
	if errors.Is(err, fs.ErrNotExist) || !ok {
		base = Source{
			Remote: true,
		}
	} else if err != nil {
		return nil, err
	} else {
		_ = f.Close()
		root, err := lookupRoot(name)
		if err != nil {
			return nil, err
		}
		base = Source{
			Root: root,
		}
	}

	return Resolve(ctx, &base, name, subToolName)
}

func Resolve(ctx context.Context, base *Source, name, subTool string) (*types.Tool, error) {
	s, err := Input(ctx, base, name)
	if err != nil {
		return nil, err
	}

	return ReadTool(ctx, s, subTool)
}

func Input(ctx context.Context, base *Source, name string) (*Source, error) {
	name = strings.TrimSuffix(name, Suffix)

	if !base.Remote {
		s, ok, err := loadLocal(base, name)
		if err != nil || ok {
			return s, err
		}
	}

	s, ok, err := loadGithub(ctx, base, name)
	if err != nil || ok {
		return s, err
	}

	return nil, fmt.Errorf("can not load tools %s at %s", name, base.Path)
}
