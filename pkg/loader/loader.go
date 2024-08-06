package loader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/internal"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/openapi"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/gptscript-ai/gptscript/pkg/system"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

const CacheTimeout = time.Hour

type source struct {
	// Content The content of the source
	Content []byte
	// Remote indicates that this file was loaded from a remote source (not local disk)
	Remote bool
	// Path is the path of this source used to find any relative references to this source
	Path string
	// Name is the filename of this source, it does not include the path in it
	Name string
	// Location is a string representation representing the source. It's not assume to
	// be a valid URI or URL, used primarily for display.
	Location string
	// Repo The VCS repo where this tool was found, used to clone and provide the local tool code content
	Repo *types.Repo
}

func (s source) WithRemote(remote bool) *source {
	s.Remote = remote
	return &s
}

func (s *source) String() string {
	if s.Path == "" && s.Name == "" {
		return ""
	}
	return s.Path + "/" + s.Name
}

func openFile(path string) (io.ReadCloser, bool, error) {
	f, err := internal.FS.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return f, true, nil
}

func loadLocal(base *source, name string) (*source, bool, error) {
	filePath := name
	if !filepath.IsAbs(name) {
		// We want to keep all strings in / format, and only convert to platform specific when reading
		// This is why we use path instead of filepath.
		filePath = path.Join(base.Path, name)
	}

	if s, err := fs.Stat(internal.FS, filepath.Clean(filePath)); err == nil && s.IsDir() {
		for _, def := range types.DefaultFiles {
			toolPath := path.Join(filePath, def)
			if s, err := fs.Stat(internal.FS, filepath.Clean(toolPath)); err == nil && !s.IsDir() {
				filePath = toolPath
				break
			}
		}
	}

	content, ok, err := openFile(filepath.Clean(filePath))
	if err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	log.Debugf("opened %s", filePath)

	defer content.Close()

	data, err := io.ReadAll(content)
	if err != nil {
		return nil, false, err
	}

	return &source{
		Content:  data,
		Remote:   false,
		Path:     path.Dir(filePath),
		Name:     path.Base(filePath),
		Location: filePath,
	}, true, nil
}

func loadProgram(data []byte, into *types.Program, targetToolName string) (types.Tool, error) {
	var ext types.Program

	if err := json.Unmarshal(data[len(assemble.Header):], &ext); err != nil {
		return types.Tool{}, err
	}

	into.ToolSet = make(map[string]types.Tool, len(ext.ToolSet))
	for k, v := range ext.ToolSet {
		if builtinTool, ok := builtin.Builtin(k); ok {
			v = builtinTool
		}
		into.ToolSet[k] = v
	}

	tool := into.ToolSet[ext.EntryToolID]
	if targetToolName == "" {
		return tool, nil
	}

	tool, ok := into.ToolSet[tool.LocalTools[strings.ToLower(targetToolName)]]
	if !ok {
		return tool, &types.ErrToolNotFound{
			ToolName: targetToolName,
		}
	}

	return tool, nil
}

func loadOpenAPI(prg *types.Program, data []byte) *openapi3.T {
	var (
		openAPICacheKey     = hash.Digest(data)
		openAPIDocument, ok = prg.OpenAPICache[openAPICacheKey].(*openapi3.T)
		err                 error
	)

	if ok {
		return openAPIDocument
	}

	if prg.OpenAPICache == nil {
		prg.OpenAPICache = map[string]any{}
	}

	openAPIDocument, err = openapi.LoadFromBytes(data)
	if err != nil {
		return nil
	}

	prg.OpenAPICache[openAPICacheKey] = openAPIDocument
	return openAPIDocument
}

func readTool(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, targetToolName string) ([]types.Tool, error) {
	data := base.Content

	if bytes.HasPrefix(data, assemble.Header) {
		tool, err := loadProgram(data, prg, targetToolName)
		if err != nil {
			return nil, err
		}
		return []types.Tool{tool}, nil
	}

	var (
		tools     []types.Tool
		isOpenAPI bool
	)

	if openAPIDocument := loadOpenAPI(prg, data); openAPIDocument != nil {
		isOpenAPI = true
		var err error
		if base.Remote {
			tools, err = getOpenAPITools(openAPIDocument, base.Location, base.Location, targetToolName)
		} else {
			tools, err = getOpenAPITools(openAPIDocument, "", base.Name, targetToolName)
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing OpenAPI definition: %w", err)
		}
	}

	if ext := path.Ext(base.Name); len(tools) == 0 && ext != "" && ext != system.Suffix && utf8.Valid(data) {
		tools = []types.Tool{
			{
				ToolDef: types.ToolDef{
					Parameters: types.Parameters{
						Name: base.Name,
					},
					Instructions: types.EchoPrefix + "\n" + string(data),
				},
			},
		}
	}

	// If we didn't get any tools from trying to parse it as OpenAPI, try to parse it as a GPTScript
	if len(tools) == 0 {
		var err error
		tools, err = parser.ParseTools(bytes.NewReader(data), parser.Options{
			AssignGlobals: true,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools found in %s", base)
	}

	var (
		localTools  = types.ToolSet{}
		targetTools []types.Tool
	)

	for i, tool := range tools {
		tool.WorkingDir = base.Path
		tool.Source.Location = base.Location
		tool.Source.Repo = base.Repo

		// Probably a better way to come up with an ID
		tool.ID = tool.Source.Location + ":" + tool.Name

		if i != 0 && tool.Parameters.Name == "" {
			return nil, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have no name"))
		}

		if i != 0 && tool.Parameters.GlobalModelName != "" {
			return nil, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have global model name"))
		}

		if i != 0 && len(tool.Parameters.GlobalTools) > 0 {
			return nil, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have global tools"))
		}

		// Determine targetTools
		if isOpenAPI && os.Getenv("GPTSCRIPT_OPENAPI_REVAMP") == "true" {
			targetTools = append(targetTools, tool)
		} else {
			if i == 0 && targetToolName == "" {
				targetTools = append(targetTools, tool)
			}

			if targetToolName != "" && tool.Parameters.Name != "" {
				if strings.EqualFold(tool.Parameters.Name, targetToolName) {
					targetTools = append(targetTools, tool)
				} else if strings.Contains(targetToolName, "*") {
					var patterns []string
					if strings.Contains(targetToolName, "|") {
						patterns = strings.Split(targetToolName, "|")
					} else {
						patterns = []string{targetToolName}
					}

					for _, pattern := range patterns {
						match, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(tool.Parameters.Name))
						if err != nil {
							return nil, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, err)
						}
						if match {
							targetTools = append(targetTools, tool)
							break
						}
					}
				}
			}
		}

		if existing, ok := localTools[strings.ToLower(tool.Parameters.Name)]; ok {
			return nil, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo,
				fmt.Errorf("duplicate tool name [%s] in %s found at lines %d and %d", tool.Parameters.Name, tool.Source.Location,
					tool.Source.LineNo, existing.Source.LineNo))
		}

		localTools[strings.ToLower(tool.Parameters.Name)] = tool
	}

	return linkAll(ctx, cache, prg, base, targetTools, localTools)
}

func linkAll(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, tools []types.Tool, localTools types.ToolSet) (result []types.Tool, _ error) {
	localToolsMapping := make(map[string]string, len(tools))
	for _, localTool := range localTools {
		localToolsMapping[strings.ToLower(localTool.Parameters.Name)] = localTool.ID
	}

	for _, tool := range tools {
		tool, err := link(ctx, cache, prg, base, tool, localTools, localToolsMapping)
		if err != nil {
			return nil, err
		}
		result = append(result, tool)
	}
	return
}

func link(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, tool types.Tool, localTools types.ToolSet, localToolsMapping map[string]string) (types.Tool, error) {
	if existing, ok := prg.ToolSet[tool.ID]; ok {
		return existing, nil
	}

	tool.ToolMapping = map[string][]types.ToolReference{}
	tool.LocalTools = map[string]string{}
	toolNames := map[string]struct{}{}

	// Add now to break circular loops, but later we will update this tool and copy the new
	// tool to the prg.ToolSet
	prg.ToolSet[tool.ID] = tool

	// The below is done in two loops so that local names stay as the tool names
	// and don't get mangled by external references

	for _, targetToolName := range tool.Parameters.ToolRefNames() {
		noArgs, _ := types.SplitArg(targetToolName)
		localTool, ok := localTools[strings.ToLower(noArgs)]
		if ok {
			var linkedTool types.Tool
			if existing, ok := prg.ToolSet[localTool.ID]; ok {
				linkedTool = existing
			} else {
				var err error
				linkedTool, err = link(ctx, cache, prg, base, localTool, localTools, localToolsMapping)
				if err != nil {
					return types.Tool{}, fmt.Errorf("failed linking %s at %s: %w", targetToolName, base, err)
				}
			}

			tool.AddToolMapping(targetToolName, linkedTool)
			toolNames[targetToolName] = struct{}{}
		} else {
			toolName, subTool := types.SplitToolRef(targetToolName)
			resolvedTools, err := resolve(ctx, cache, prg, base, toolName, subTool)
			if err != nil {
				return types.Tool{}, fmt.Errorf("failed resolving %s from %s: %w", targetToolName, base, err)
			}
			for _, resolvedTool := range resolvedTools {
				tool.AddToolMapping(targetToolName, resolvedTool)
			}
		}
	}

	tool.LocalTools = localToolsMapping

	tool = builtin.SetDefaults(tool)
	prg.ToolSet[tool.ID] = tool

	return tool, nil
}

func ProgramFromSource(ctx context.Context, content, subToolName string, opts ...Options) (types.Program, error) {
	if log.IsDebug() {
		start := time.Now()
		defer func() {
			log.Debugf("loaded program from source took %v", time.Since(start))
		}()
	}
	opt := complete(opts...)

	var locationPath, locationName string
	if opt.Location != "" {
		locationPath = path.Dir(opt.Location)
		locationName = path.Base(opt.Location)
	}

	prg := types.Program{
		ToolSet: types.ToolSet{},
	}
	tools, err := readTool(ctx, opt.Cache, &prg, &source{
		Content:  []byte(content),
		Path:     locationPath,
		Name:     locationName,
		Location: opt.Location,
	}, subToolName)
	if err != nil {
		return types.Program{}, err
	}
	prg.EntryToolID = tools[0].ID
	return prg, nil
}

type Options struct {
	Cache    *cache.Client
	Location string
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.Cache = types.FirstSet(opt.Cache, result.Cache)
		result.Location = types.FirstSet(opt.Location, result.Location)
	}

	if result.Location == "" {
		result.Location = "inline"
	}

	return
}

func Program(ctx context.Context, name, subToolName string, opts ...Options) (types.Program, error) {
	// We want all paths to have / not \
	name = strings.ReplaceAll(name, "\\", "/")

	if log.IsDebug() {
		start := time.Now()
		defer func() {
			log.Debugf("loaded program %s source took %v", name, time.Since(start))
		}()
	}

	opt := complete(opts...)

	if subToolName == "" {
		name, subToolName = types.SplitToolRef(name)
	}
	prg := types.Program{
		Name:    name,
		ToolSet: types.ToolSet{},
	}
	tools, err := resolve(ctx, opt.Cache, &prg, &source{}, name, subToolName)
	if err != nil {
		return types.Program{}, err
	}
	prg.EntryToolID = tools[0].ID
	return prg, nil
}

func resolve(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, name, subTool string) ([]types.Tool, error) {
	if subTool == "" {
		t, ok := builtin.Builtin(name)
		if ok {
			prg.ToolSet[t.ID] = t
			return []types.Tool{t}, nil
		}
	}

	s, err := input(ctx, cache, base, name)
	if err != nil {
		return nil, err
	}

	result, err := readTool(ctx, cache, prg, s, subTool)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, types.NewErrToolNotFound(types.ToToolName(name, subTool))
	}

	return result, nil
}

func input(ctx context.Context, cache *cache.Client, base *source, name string) (*source, error) {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		// copy and modify
		base = base.WithRemote(true)
	}

	if !base.Remote {
		s, ok, err := loadLocal(base, name)
		if err != nil || ok {
			return s, err
		}
	}

	s, ok, err := loadURL(ctx, cache, base, name)
	if err != nil || ok {
		return s, err
	}

	return nil, fmt.Errorf("can not load tools path=%s name=%s", base.Path, name)
}
