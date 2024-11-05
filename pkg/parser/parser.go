package parser

import (
	"fmt"
	"io"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	sepRegex       = regexp.MustCompile(`^\s*---+\s*$`)
	endHeaderRegex = regexp.MustCompile(`^\s*===+\s*$`)
	strictSepRegex = regexp.MustCompile(`^---\n$`)
	skipRegex      = regexp.MustCompile(`^![ -.:*\w]+\s*$`)
	nameRegex      = regexp.MustCompile(`^[a-z]+$`)
)

func normalize(key string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(key, " ", "")))
}

func toBool(line string) (bool, error) {
	line = normalize(line)
	if line == "true" || line == "t" {
		return true, nil
	} else if line != "false" {
		return false, fmt.Errorf("invalid boolean parameter, must be \"true\" or \"false\", got [%s]", line)
	}
	return false, nil
}

func toFloatPtr(line string) (*float32, error) {
	f, err := strconv.ParseFloat(line, 32)
	if err != nil {
		return nil, err
	}
	f32 := float32(f)
	return &f32, nil
}

func csv(line string) (result []string) {
	for _, part := range strings.Split(line, ",") {
		result = append(result, strings.TrimSpace(part))
	}
	return
}

func addArg(line string, tool *types.Tool) error {
	if tool.Parameters.Arguments == nil {
		tool.Parameters.Arguments = &openapi3.Schema{
			Type:       &openapi3.Types{"object"},
			Properties: openapi3.Schemas{},
		}
	}

	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return fmt.Errorf("invalid arg format: %s", line)
	}

	tool.Parameters.Arguments.Properties[key] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Description: strings.TrimSpace(value),
			Type:        &openapi3.Types{"string"},
		},
	}

	return nil
}

func isParam(line string, tool *types.Tool, scan *simplescanner) (_ bool, err error) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return false, nil
	}
	value = strings.TrimSpace(value)
	switch normalize(key) {
	case "name":
		tool.Parameters.Name = value
	case "modelprovider":
		tool.Parameters.ModelProvider = true
	case "model", "modelname":
		tool.Parameters.ModelName = value
	case "globalmodel", "globalmodelname":
		tool.Parameters.GlobalModelName = value
	case "description":
		tool.Parameters.Description = scan.AddMultiline(value)
	case "internalprompt":
		v, err := toBool(value)
		if err != nil {
			return false, err
		}
		tool.Parameters.InternalPrompt = &v
	case "chat":
		v, err := toBool(value)
		if err != nil {
			return false, err
		}
		tool.Parameters.Chat = v
	case "export", "exporttool", "exports", "exporttools", "sharetool", "sharetools", "sharedtool", "sharedtools":
		tool.Parameters.Export = append(tool.Parameters.Export, csv(scan.AddMultiline(value))...)
	case "tool", "tools":
		tool.Parameters.Tools = append(tool.Parameters.Tools, csv(scan.AddMultiline(value))...)
	case "inputfilter", "inputfilters":
		tool.Parameters.InputFilters = append(tool.Parameters.InputFilters, csv(scan.AddMultiline(value))...)
	case "shareinputfilter", "shareinputfilters", "sharedinputfilter", "sharedinputfilters":
		tool.Parameters.ExportInputFilters = append(tool.Parameters.ExportInputFilters, csv(scan.AddMultiline(value))...)
	case "outputfilter", "outputfilters":
		tool.Parameters.OutputFilters = append(tool.Parameters.OutputFilters, csv(scan.AddMultiline(value))...)
	case "shareoutputfilter", "shareoutputfilters", "sharedoutputfilter", "sharedoutputfilters":
		tool.Parameters.ExportOutputFilters = append(tool.Parameters.ExportOutputFilters, csv(scan.AddMultiline(value))...)
	case "agent", "agents":
		tool.Parameters.Agents = append(tool.Parameters.Agents, csv(scan.AddMultiline(value))...)
	case "globaltool", "globaltools":
		tool.Parameters.GlobalTools = append(tool.Parameters.GlobalTools, csv(scan.AddMultiline(value))...)
	case "exportcontext", "exportcontexts", "sharecontext", "sharecontexts", "sharedcontext", "sharedcontexts":
		tool.Parameters.ExportContext = append(tool.Parameters.ExportContext, csv(scan.AddMultiline(value))...)
	case "context":
		tool.Parameters.Context = append(tool.Parameters.Context, csv(scan.AddMultiline(value))...)
	case "stdin":
		b, err := toBool(value)
		if err != nil {
			return false, err
		}
		tool.Parameters.Stdin = b
	case "metadata":
		mkey, mvalue, _ := strings.Cut(scan.AddMultiline(value), ":")
		if tool.MetaData == nil {
			tool.MetaData = map[string]string{}
		}
		tool.MetaData[strings.TrimSpace(mkey)] = strings.TrimSpace(mvalue)
	case "args", "arg", "param", "params", "parameters", "parameter":
		if err := addArg(scan.AddMultiline(value), tool); err != nil {
			return false, err
		}
	case "maxtoken", "maxtokens":
		tool.Parameters.MaxTokens, err = strconv.Atoi(value)
		if err != nil {
			return false, err
		}
	case "cache":
		b, err := toBool(value)
		if err != nil {
			return false, err
		}
		tool.Parameters.Cache = &b
	case "jsonmode", "json", "jsonoutput", "jsonformat", "jsonresponse":
		tool.Parameters.JSONResponse, err = toBool(value)
		if err != nil {
			return false, err
		}
	case "temperature":
		tool.Parameters.Temperature, err = toFloatPtr(value)
		if err != nil {
			return false, err
		}
	case "credentials", "creds", "credential", "cred":
		tool.Parameters.Credentials = append(tool.Parameters.Credentials, csv(scan.AddMultiline(value))...)
	case "sharecredentials", "sharecreds", "sharecredential", "sharecred", "sharedcredentials", "sharedcreds", "sharedcredential", "sharedcred":
		tool.Parameters.ExportCredentials = append(tool.Parameters.ExportCredentials, scan.AddMultiline(value))
	case "type":
		tool.Type = types.ToolType(strings.ToLower(value))
	default:
		return nameRegex.MatchString(key), nil
	}

	return true, nil
}

type ErrLine struct {
	Path string
	Line int
	Err  error
}

func (e *ErrLine) Unwrap() error {
	return e.Err
}

func (e *ErrLine) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("line %d: %v", e.Line, e.Err)
	}
	return fmt.Sprintf("line %s:%d: %v", e.Path, e.Line, e.Err)
}

func NewErrLine(path string, lineNo int, err error) error {
	return &ErrLine{
		Path: path,
		Line: lineNo,
		Err:  err,
	}
}

type context struct {
	tool         types.Tool
	instructions []string
	inBody       bool
	skipNode     bool
	skipLines    []string
	seenParam    bool
}

func (c *context) finish(tools *[]Node) {
	c.tool.Instructions = strings.TrimSpace(strings.Join(c.instructions, ""))
	if c.tool.Instructions != "" ||
		c.tool.Parameters.Name != "" ||
		len(c.tool.Export) > 0 ||
		len(c.tool.Tools) > 0 ||
		c.tool.GlobalModelName != "" ||
		len(c.tool.GlobalTools) > 0 ||
		len(c.tool.ExportInputFilters) > 0 ||
		len(c.tool.ExportOutputFilters) > 0 ||
		len(c.tool.Agents) > 0 ||
		len(c.tool.ExportCredentials) > 0 ||
		c.tool.Chat {
		*tools = append(*tools, Node{
			ToolNode: &ToolNode{
				Tool: c.tool,
			},
		})
	}
	if c.skipNode && len(c.skipLines) > 0 {
		*tools = append(*tools, Node{
			TextNode: &TextNode{
				Text: strings.Join(c.skipLines, ""),
			},
		})
	}
	*c = context{}
}

type Options struct {
	AssignGlobals bool
	Location      string
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.AssignGlobals = types.FirstSet(opt.AssignGlobals, result.AssignGlobals)
		result.Location = types.FirstSet(opt.Location, result.Location)
	}
	return
}

type Document struct {
	Nodes []Node `json:"nodes,omitempty"`
}

func writeSep(buf *strings.Builder, lastText bool) {
	if buf.Len() > 0 {
		if !lastText {
			buf.WriteString("\n")
		}
		buf.WriteString("---\n")
	}
}

func (d Document) String() string {
	buf := strings.Builder{}
	lastText := false
	for _, node := range d.Nodes {
		if node.TextNode != nil {
			writeSep(&buf, lastText)
			buf.WriteString(node.TextNode.Text)
			lastText = true
		}
		if node.ToolNode != nil {
			writeSep(&buf, lastText)
			buf.WriteString(node.ToolNode.Tool.String())
			lastText = false
		}
	}
	return buf.String()
}

type Node struct {
	TextNode *TextNode `json:"textNode,omitempty"`
	ToolNode *ToolNode `json:"toolNode,omitempty"`
}

type TextNode struct {
	Text string `json:"text,omitempty"`
}

type ToolNode struct {
	Tool types.Tool `json:"tool,omitempty"`
}

func ParseTools(input io.Reader, opts ...Options) (result []types.Tool, _ error) {
	doc, err := Parse(input, opts...)
	if err != nil {
		return nil, err
	}
	for _, node := range doc.Nodes {
		if node.ToolNode != nil {
			result = append(result, node.ToolNode.Tool)
		}
	}

	return
}

func Parse(input io.Reader, opts ...Options) (Document, error) {
	nodes, err := parse(input)
	if err != nil {
		return Document{}, err
	}

	opt := complete(opts...)

	if opt.Location != "" {
		for _, node := range nodes {
			if node.ToolNode != nil && node.ToolNode.Tool.Source.Location == "" {
				node.ToolNode.Tool.Source.Location = opt.Location
			}
		}
	}

	nodes = assignMetadata(nodes)

	if !opt.AssignGlobals {
		return Document{
			Nodes: nodes,
		}, nil
	}

	var (
		globalModel     string
		seenGlobalTools = map[string]struct{}{}
		globalTools     []string
	)

	for _, node := range nodes {
		if node.ToolNode == nil {
			continue
		}
		tool := node.ToolNode.Tool
		if tool.GlobalModelName != "" {
			if globalModel != "" {
				return Document{}, fmt.Errorf("global model name defined multiple times")
			}
			globalModel = tool.GlobalModelName
		}
		for _, globalTool := range tool.GlobalTools {
			if _, ok := seenGlobalTools[globalTool]; ok {
				continue
			}
			seenGlobalTools[globalTool] = struct{}{}
			globalTools = append(globalTools, globalTool)
		}
	}

	for _, node := range nodes {
		if node.ToolNode == nil {
			continue
		}
		if globalModel != "" && node.ToolNode.Tool.ModelName == "" {
			node.ToolNode.Tool.ModelName = globalModel
		}
		for _, globalTool := range globalTools {
			if !slices.Contains(node.ToolNode.Tool.Tools, globalTool) {
				node.ToolNode.Tool.Tools = append(node.ToolNode.Tool.Tools, globalTool)
			}
		}
	}

	return Document{
		Nodes: nodes,
	}, nil
}

func assignMetadata(nodes []Node) (result []Node) {
	metadata := map[string]map[string]string{}
	result = make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if node.TextNode != nil {
			body, ok := strings.CutPrefix(node.TextNode.Text, "!metadata:")
			if ok {
				line, rest, ok := strings.Cut(body, "\n")
				if ok {
					toolName, metaKey, ok := strings.Cut(strings.TrimSpace(line), ":")
					if ok {
						d, ok := metadata[toolName]
						if !ok {
							d = map[string]string{}
							metadata[toolName] = d
						}
						d[metaKey] = strings.TrimSpace(rest)
					}
				}
			}
		}
	}
	if len(metadata) == 0 {
		return nodes
	}

	for _, node := range nodes {
		if node.ToolNode != nil {
			if node.ToolNode.Tool.MetaData == nil {
				node.ToolNode.Tool.MetaData = map[string]string{}
			}
			maps.Copy(node.ToolNode.Tool.MetaData, metadata[node.ToolNode.Tool.Name])
			for wildcard := range metadata {
				if strings.Contains(wildcard, "*") {
					if m, err := path.Match(wildcard, node.ToolNode.Tool.Name); m && err == nil {
						if node.ToolNode.Tool.MetaData == nil {
							node.ToolNode.Tool.MetaData = map[string]string{}
						}
						maps.Copy(node.ToolNode.Tool.MetaData, metadata[wildcard])
					}
				}
			}
		}
		result = append(result, node)
	}

	return
}

func isGPTScriptHashBang(line string) bool {
	if !strings.HasPrefix(line, "#!") {
		return false
	}

	parts := strings.Fields(line)

	// Very specific lines we are looking for
	// 1. #!gptscript
	// 2. #!/usr/bin/env gptscript
	// 3. #!/bin/env gptscript

	if parts[0] == "#!gptscript" {
		return true
	}

	if len(parts) > 1 && (parts[0] == "#!/usr/bin/env" || parts[0] == "#!/bin/env") &&
		parts[1] == "gptscript" {
		return true
	}

	return false
}

type simplescanner struct {
	lines []string
}

func newSimpleScanner(data []byte) *simplescanner {
	if len(data) == 0 {
		return &simplescanner{}
	}
	lines := strings.Split(string(data), "\n")
	return &simplescanner{
		lines: append([]string{""}, lines...),
	}
}

func dropCR(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\r' {
		return s[:len(s)-1]
	}
	return s
}

func (s *simplescanner) AddMultiline(current string) string {
	result := current
	for {
		if len(s.lines) < 2 || len(s.lines[1]) == 0 {
			return result
		}
		if strings.HasPrefix(s.lines[1], " ") || strings.HasPrefix(s.lines[1], "\t") {
			result += " " + dropCR(s.lines[1])
			s.lines = s.lines[1:]
		} else {
			return result
		}
	}
}

func (s *simplescanner) Text() string {
	if len(s.lines) == 0 {
		return ""
	}
	return dropCR(s.lines[0])
}

func (s *simplescanner) Scan() bool {
	if len(s.lines) == 0 {
		return false
	}
	s.lines = s.lines[1:]
	return true
}

func parse(input io.Reader) ([]Node, error) {
	var (
		tools   []Node
		context context
		lineNo  int
	)

	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	scan := newSimpleScanner(data)

	for scan.Scan() {
		lineNo++
		if context.tool.Source.LineNo == 0 {
			context.tool.Source.LineNo = lineNo
		}

		line := scan.Text() + "\n"

		if context.skipNode {
			if strictSepRegex.MatchString(line) {
				context.finish(&tools)
				continue
			}
		} else if sepRegex.MatchString(line) {
			context.finish(&tools)
			continue
		}

		if context.skipNode {
			context.skipLines = append(context.skipLines, line)
			continue
		}

		if !context.inBody {
			// If the very first line is #! just skip because this is a unix interpreter declaration
			if lineNo == 1 && isGPTScriptHashBang(line) {
				continue
			}

			// This is a comment
			if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#!") {
				continue
			}

			if !context.seenParam && skipRegex.MatchString(line) {
				context.skipLines = append(context.skipLines, line)
				context.skipNode = true
				continue
			}

			// Blank line
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Look for params
			if isParam, err := isParam(line, &context.tool, scan); err != nil {
				return nil, NewErrLine("", lineNo, err)
			} else if isParam {
				context.seenParam = true
				continue
			} else if endHeaderRegex.MatchString(line) {
				// force the end of the header and don't include the current line in the header
				context.inBody = true
				continue
			}
		}

		context.inBody = true
		context.instructions = append(context.instructions, line)
	}

	context.finish(&tools)
	return tools, nil
}
