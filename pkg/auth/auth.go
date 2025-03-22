package auth

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/gptscript-ai/gptscript/pkg/builtin"
	"github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/runner"
)

func Authorize(ctx engine.Context, input string) (runner.AuthorizerResponse, error) {
	defer context.GetPauseFuncFromCtx(ctx.Ctx)()()

	if IsSafe(ctx) {
		return runner.AuthorizerResponse{
			Accept: true,
		}, nil
	}

	var result bool
	err := survey.AskOne(&survey.Confirm{
		Help:    fmt.Sprintf("The full source of the tools is as follows:\n\n%s", ctx.Tool.Print()),
		Default: true,
		Message: ConfirmMessage(ctx, input),
	}, &result)
	if err != nil {
		return runner.AuthorizerResponse{}, err
	}

	return runner.AuthorizerResponse{
		Accept:  result,
		Message: "Request denied, blocking execution.",
	}, nil
}

func IsSafe(ctx engine.Context) bool {
	if !ctx.Tool.IsCommand() {
		return true
	}

	_, ok := builtin.SafeTools[strings.Split(ctx.Tool.Instructions, "\n")[0][2:]]
	return ok
}

func ConfirmMessage(ctx engine.Context, input string) string {
	var (
		loc         = ctx.Tool.Source.Location
		interpreter = strings.Split(ctx.Tool.Instructions, "\n")[0][2:]
	)

	if ctx.Tool.Source.Repo != nil {
		loc = ctx.Tool.Source.Repo.Root
		loc = strings.TrimPrefix(loc, "https://")
		loc = strings.TrimSuffix(loc, ".git")
		loc = filepath.Join(loc, ctx.Tool.Source.Repo.Path, ctx.Tool.Source.Repo.Name)
	}

	if ctx.Tool.BuiltinFunc != nil {
		loc = "Builtin"
	}

	return fmt.Sprintf(`Description: %s
  Interpreter: %s
  Source: %s
  Input: %s
Allow the above tool to execute?`, ctx.Tool.Description, interpreter, loc, strings.TrimSpace(input))
}
