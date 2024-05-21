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
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/getkin/kin-openapi/openapi2"

	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/assemble"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/parser"
	"github.com/gptscript-ai/gptscript/pkg/system"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"
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

	if s, err := os.Stat(path); err == nil && s.IsDir() {
		toolPath := filepath.Join(base.Path, name, "tool.gpt")
		if s, err := os.Stat(toolPath); err == nil && !s.IsDir() {
			path = toolPath
		}
	}

	content, ok, err := openFile(path)
	if err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}
	log.Debugf("opened %s", path)

	defer content.Close()

	data, err := io.ReadAll(content)
	if err != nil {
		return nil, false, err
	}

	return &source{
		Content:  data,
		Remote:   false,
		Path:     filepath.Dir(path),
		Name:     filepath.Base(path),
		Location: path,
	}, true, nil
}

func loadProgram(data []byte, into *types.Program, targetToolName string) (types.Tool, error) {
	var (
		ext types.Program
	)

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

	switch isOpenAPI(data) {
	case 2:
		// Convert OpenAPI v2 to v3
		jsondata := data
		if !json.Valid(data) {
			jsondata, err = kyaml.YAMLToJSON(data)
			if err != nil {
				return nil
			}
		}

		doc := &openapi2.T{}
		if err := doc.UnmarshalJSON(jsondata); err != nil {
			return nil
		}

		openAPIDocument, err = openapi2conv.ToV3(doc)
		if err != nil {
			return nil
		}
	case 3:
		// Use OpenAPI v3 as is
		openAPIDocument, err = openapi3.NewLoader().LoadFromData(data)
		if err != nil {
			return nil
		}
	default:
		return nil
	}

	prg.OpenAPICache[openAPICacheKey] = openAPIDocument
	return openAPIDocument
}

func readTool(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, targetToolName string) (types.Tool, error) {
	data := base.Content

	if bytes.HasPrefix(data, assemble.Header) {
		return loadProgram(data, prg, targetToolName)
	}

	var (
		tools []types.Tool
	)

	if openAPIDocument := loadOpenAPI(prg, data); openAPIDocument != nil {
		var err error
		if base.Remote {
			tools, err = getOpenAPITools(openAPIDocument, base.Location)
		} else {
			tools, err = getOpenAPITools(openAPIDocument, "")
		}
		if err != nil {
			return types.Tool{}, fmt.Errorf("error parsing OpenAPI definition: %w", err)
		}
	}

	if ext := path.Ext(base.Name); len(tools) == 0 && ext != "" && ext != system.Suffix && utf8.Valid(data) {
		tools = []types.Tool{
			{
				Parameters: types.Parameters{
					Name: base.Name,
				},
				Instructions: types.EchoPrefix + "\n" + string(data),
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
			return types.Tool{}, err
		}
	}

	if len(tools) == 0 {
		return types.Tool{}, fmt.Errorf("no tools found in %s", base)
	}

	var (
		localTools = types.ToolSet{}
		mainTool   types.Tool
	)

	for i, tool := range tools {
		tool.WorkingDir = base.Path
		tool.Source.Location = base.Location
		tool.Source.Repo = base.Repo

		// Probably a better way to come up with an ID
		tool.ID = tool.Source.Location + ":" + tool.Name

		if i == 0 {
			mainTool = tool
		}

		if i != 0 && tool.Parameters.Name == "" {
			return types.Tool{}, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have no name"))
		}

		if i != 0 && tool.Parameters.GlobalModelName != "" {
			return types.Tool{}, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have global model name"))
		}

		if i != 0 && len(tool.Parameters.GlobalTools) > 0 {
			return types.Tool{}, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo, fmt.Errorf("only the first tool in a file can have global tools"))
		}

		if targetToolName != "" && strings.EqualFold(tool.Parameters.Name, targetToolName) {
			mainTool = tool
		}

		if existing, ok := localTools[strings.ToLower(tool.Parameters.Name)]; ok {
			return types.Tool{}, parser.NewErrLine(tool.Source.Location, tool.Source.LineNo,
				fmt.Errorf("duplicate tool name [%s] in %s found at lines %d and %d", tool.Parameters.Name, tool.Source.Location,
					tool.Source.LineNo, existing.Source.LineNo))
		}

		localTools[strings.ToLower(tool.Parameters.Name)] = tool
	}

	return link(ctx, cache, prg, base, mainTool, localTools)
}

func link(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, tool types.Tool, localTools types.ToolSet) (types.Tool, error) {
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

	for _, targetToolName := range slices.Concat(tool.Parameters.Tools,
		tool.Parameters.Export,
		tool.Parameters.ExportContext,
		tool.Parameters.Context,
		tool.Parameters.Credentials) {
		noArgs, _ := types.SplitArg(targetToolName)
		localTool, ok := localTools[strings.ToLower(noArgs)]
		if ok {
			var linkedTool types.Tool
			if existing, ok := prg.ToolSet[localTool.ID]; ok {
				linkedTool = existing
			} else {
				var err error
				linkedTool, err = link(ctx, cache, prg, base, localTool, localTools)
				if err != nil {
					return types.Tool{}, fmt.Errorf("failed linking %s at %s: %w", targetToolName, base, err)
				}
			}

			tool.ToolMapping[targetToolName] = linkedTool.ID
			toolNames[targetToolName] = struct{}{}
		} else {
			toolName, subTool := types.SplitToolRef(targetToolName)
			resolvedTool, err := resolve(ctx, cache, prg, base, toolName, subTool)
			if err != nil {
				return types.Tool{}, fmt.Errorf("failed resolving %s from %s: %w", targetToolName, base, err)
			}

			tool.ToolMapping[targetToolName] = resolvedTool.ID
		}
	}

	for _, localTool := range localTools {
		tool.LocalTools[strings.ToLower(localTool.Parameters.Name)] = localTool.ID
	}

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

	prg := types.Program{
		ToolSet: types.ToolSet{},
	}
	tool, err := readTool(ctx, opt.Cache, &prg, &source{
		Content:  []byte(content),
		Location: "inline",
	}, subToolName)
	if err != nil {
		return types.Program{}, err
	}
	prg.EntryToolID = tool.ID
	return prg, nil
}

type Options struct {
	Cache *cache.Client
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.Cache = types.FirstSet(opt.Cache, result.Cache)
	}

	return
}

func Program(ctx context.Context, name, subToolName string, opts ...Options) (types.Program, error) {
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
	tool, err := resolve(ctx, opt.Cache, &prg, &source{}, name, subToolName)
	if err != nil {
		return types.Program{}, err
	}
	prg.EntryToolID = tool.ID
	return prg, nil
}

func resolve(ctx context.Context, cache *cache.Client, prg *types.Program, base *source, name, subTool string) (types.Tool, error) {
	if subTool == "" {
		t, ok := builtin.Builtin(name)
		if ok {
			prg.ToolSet[t.ID] = t
			return t, nil
		}
	}

	s, err := input(ctx, cache, base, name)
	if err != nil {
		return types.Tool{}, err
	}

	return readTool(ctx, cache, prg, s, subTool)
}

func input(ctx context.Context, cache *cache.Client, base *source, name string) (*source, error) {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		base.Remote = true
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

// isOpenAPI checks if the data is an OpenAPI definition and returns the version if it is.
func isOpenAPI(data []byte) int {
	var fragment struct {
		Paths   map[string]any `json:"paths,omitempty"`
		Swagger string         `json:"swagger,omitempty"`
		OpenAPI string         `json:"openapi,omitempty"`
	}

	if err := json.Unmarshal(data, &fragment); err != nil {
		if err := yaml.Unmarshal(data, &fragment); err != nil {
			return 0
		}
	}
	if len(fragment.Paths) == 0 {
		return 0
	}

	if v, _, _ := strings.Cut(fragment.OpenAPI, "."); v != "" {
		ver, err := strconv.Atoi(v)
		if err != nil {
			log.Debugf("invalid OpenAPI version: openapi=%q", fragment.OpenAPI)
			return 0
		}
		return ver
	}

	if v, _, _ := strings.Cut(fragment.Swagger, "."); v != "" {
		ver, err := strconv.Atoi(v)
		if err != nil {
			log.Debugf("invalid Swagger version: swagger=%q", fragment.Swagger)
			return 0
		}
		return ver
	}

	log.Debugf("no OpenAPI version found in input data: openapi=%q, swagger=%q", fragment.OpenAPI, fragment.Swagger)
	return 0
}
