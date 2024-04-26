package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	sepRegex       = regexp.MustCompile(`^\s*---+\s*$`)
	strictSepRegex = regexp.MustCompile(`^---\n$`)
	skipRegex      = regexp.MustCompile(`^![-\w]+\s*$`)
)

func normalize(key string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(key, " ", "")))
}

func toBool(line string) (bool, error) {
	if line == "true" {
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
			Type:       "object",
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
			Type:        "string",
		},
	}

	return nil
}

func isParam(line string, tool *types.Tool) (_ bool, err error) {
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
		tool.Parameters.Description = value
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
	case "export":
		tool.Parameters.Export = append(tool.Parameters.Export, csv(strings.ToLower(value))...)
	case "tool", "tools":
		tool.Parameters.Tools = append(tool.Parameters.Tools, csv(strings.ToLower(value))...)
	case "globaltool", "globaltools":
		tool.Parameters.GlobalTools = append(tool.Parameters.GlobalTools, csv(strings.ToLower(value))...)
	case "exportcontext":
		tool.Parameters.ExportContext = append(tool.Parameters.ExportContext, csv(strings.ToLower(value))...)
	case "context":
		tool.Parameters.Context = append(tool.Parameters.Context, csv(strings.ToLower(value))...)
	case "args", "arg", "param", "params", "parameters", "parameter":
		if err := addArg(value, tool); err != nil {
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
		tool.Parameters.Credentials = append(tool.Parameters.Credentials, csv(strings.ToLower(value))...)
	default:
		return false, nil
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
	seenParam    bool
}

func (c *context) finish(tools *[]types.Tool) {
	c.tool.Instructions = strings.TrimSpace(strings.Join(c.instructions, ""))
	if c.tool.Instructions != "" || c.tool.Parameters.Name != "" ||
		len(c.tool.Export) > 0 || len(c.tool.Tools) > 0 ||
		c.tool.GlobalModelName != "" ||
		len(c.tool.GlobalTools) > 0 ||
		c.tool.Chat {
		*tools = append(*tools, c.tool)
	}
	*c = context{}
}

type Options struct {
	AssignGlobals bool
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.AssignGlobals = types.FirstSet(opt.AssignGlobals, result.AssignGlobals)
	}
	return
}

func Parse(input io.Reader, opts ...Options) ([]types.Tool, error) {
	tools, err := parse(input)
	if err != nil {
		return nil, err
	}

	opt := complete(opts...)

	if !opt.AssignGlobals {
		return tools, nil
	}

	var (
		globalModel     string
		seenGlobalTools = map[string]struct{}{}
		globalTools     []string
	)

	for _, tool := range tools {
		if tool.GlobalModelName != "" {
			if globalModel != "" {
				return nil, fmt.Errorf("global model name defined multiple times")
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

	for i, tool := range tools {
		if globalModel != "" && tool.ModelName == "" {
			tool.ModelName = globalModel
		}
		for _, globalTool := range globalTools {
			if !slices.Contains(tool.Tools, globalTool) {
				tool.Tools = append(tool.Tools, globalTool)
			}
		}
		tools[i] = tool
	}

	return tools, nil
}

func parse(input io.Reader) ([]types.Tool, error) {
	scan := bufio.NewScanner(input)

	var (
		tools   []types.Tool
		context context
		lineNo  int
	)

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
			continue
		}

		if !context.inBody {
			// If the very first line is #! just skip because this is a unix interpreter declaration
			if strings.HasPrefix(line, "#!") && lineNo == 1 {
				continue
			}

			// This is a comment
			if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#!") {
				continue
			}

			if !context.seenParam && skipRegex.MatchString(line) {
				context.skipNode = true
				continue
			}

			// Blank line
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Look for params
			if isParam, err := isParam(line, &context.tool); err != nil {
				return nil, NewErrLine("", lineNo, err)
			} else if isParam {
				context.seenParam = true
				continue
			}
		}

		context.inBody = true
		context.instructions = append(context.instructions, line)
	}

	context.finish(&tools)
	return tools, nil
}
