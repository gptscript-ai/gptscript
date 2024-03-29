package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	sepRegex = regexp.MustCompile(`^\s*---+\s*$`)
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
		tool.Parameters.Arguments = &types.JSONSchema{
			Property: types.Property{
				Type: "object",
			},
			Properties: map[string]types.Property{},
		}
	}

	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return fmt.Errorf("invalid arg format: %s", line)
	}

	tool.Parameters.Arguments.Properties[key] = types.Property{
		Description: value,
		Type:        "string",
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
		tool.Parameters.Name = strings.ToLower(value)
	case "modelprovider":
		tool.Parameters.ModelProvider = true
	case "model", "modelname":
		tool.Parameters.ModelName = value
	case "description":
		tool.Parameters.Description = value
	case "internalprompt":
		v, err := toBool(value)
		if err != nil {
			return false, err
		}
		tool.Parameters.InternalPrompt = &v
	case "export":
		tool.Parameters.Export = append(tool.Parameters.Export, csv(strings.ToLower(value))...)
	case "tool", "tools":
		tool.Parameters.Tools = append(tool.Parameters.Tools, csv(strings.ToLower(value))...)
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
}

func (c *context) finish(tools *[]types.Tool) {
	c.tool.Instructions = strings.TrimSpace(strings.Join(c.instructions, ""))
	if c.tool.Instructions != "" || c.tool.Parameters.Name != "" || len(c.tool.Export) > 0 || len(c.tool.Tools) > 0 {
		*tools = append(*tools, c.tool)
	}
	*c = context{}
}

func commentEmbedded(line string) (string, bool) {
	for _, i := range []string{"#", "# ", "//", "// "} {
		prefix := i + "gptscript:"
		cut, ok := strings.CutPrefix(line, prefix)
		if ok {
			return strings.TrimSpace(cut) + "\n", ok
		}
	}
	return line, false
}

func Parse(input io.Reader) ([]types.Tool, error) {
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
		if embeddedLine, ok := commentEmbedded(line); ok {
			// Strip special comments to allow embedding the preamble in python or other interpreted languages
			line = embeddedLine
		}

		if sepRegex.MatchString(line) {
			context.finish(&tools)
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

			// Blank line
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Look for params
			if isParam, err := isParam(line, &context.tool); err != nil {
				return nil, NewErrLine("", lineNo, err)
			} else if isParam {
				continue
			}
		}

		context.inBody = true
		context.instructions = append(context.instructions, line)
	}

	context.finish(&tools)
	return tools, nil
}
