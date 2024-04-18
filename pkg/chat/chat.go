package chat

import (
	"context"

	"github.com/fatih/color"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Prompter interface {
	Readline() (string, bool, error)
	Printf(format string, args ...interface{}) (int, error)
	SetPrompt(p string)
	Close() error
}

type Chatter interface {
	Chat(ctx context.Context, prevState runner.ChatState, prg types.Program, env []string, input string) (resp runner.ChatResponse, err error)
}

type GetProgram func() (types.Program, error)

func getPrompt(prg types.Program, resp runner.ChatResponse) string {
	name := prg.ChatName()
	if newName := prg.ToolSet[resp.ToolID].Name; newName != "" {
		name = newName
	}

	return color.GreenString("%s> ", name)
}

func Start(ctx context.Context, prevState runner.ChatState, chatter Chatter, prg GetProgram, env []string, startInput string) error {
	var (
		prompter Prompter
	)

	prompter, err := newReadlinePrompter()
	if err != nil {
		return err
	}
	defer prompter.Close()

	for {
		var (
			input string
			ok    bool
			resp  runner.ChatResponse
		)

		prg, err := prg()
		if err != nil {
			return err
		}

		prompter.SetPrompt(getPrompt(prg, resp))

		if startInput != "" {
			input = startInput
			startInput = ""
		} else if !(prevState == nil && prg.ToolSet[prg.EntryToolID].Arguments == nil) {
			// The above logic will skip prompting if this is the first loop and the chat expects no args
			input, ok, err = prompter.Readline()
			if !ok || err != nil {
				return err
			}
		}

		resp, err = chatter.Chat(ctx, prevState, prg, env, input)
		if err != nil || resp.Done {
			return err
		}

		if resp.Content != "" {
			_, err := prompter.Printf(color.RedString("< %s\n", resp.Content))
			if err != nil {
				return err
			}
		}

		prevState = resp.State
	}
}
