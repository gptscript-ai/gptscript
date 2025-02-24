package chat

import (
	"context"
	"os"

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
	Chat(ctx context.Context, prevState runner.ChatState, prg types.Program, env []string, input string, opts runner.RunOptions) (resp runner.ChatResponse, err error)
}

type GetProgram func() (types.Program, error)

func getPrompt(prg types.Program, resp runner.ChatResponse) string {
	name := prg.ChatName()
	if newName := prg.ToolSet[resp.ToolID].Name; newName != "" {
		name = newName
	}

	return color.GreenString("%s> ", name)
}

func Start(ctx context.Context, prevState runner.ChatState, chatter Chatter, prg GetProgram, env []string, startInput, chatStateSaveFile string) error {
	var (
		prompter Prompter
	)

	prompter, err := newReadlinePrompter(prg)
	if err != nil {
		return err
	}
	defer prompter.Close()

	// We will want the tool name to be displayed in the prompt
	var prevResp runner.ChatResponse
	for {
		var (
			input string
			ok    bool
			resp  runner.ChatResponse
		)

		prog, err := prg()
		if err != nil {
			return err
		}

		prompter.SetPrompt(getPrompt(prog, prevResp))

		if startInput != "" {
			input = startInput
			startInput = ""
		} else if targetTool := prog.ToolSet[prog.EntryToolID]; !((prevState == nil || prevState == "") && targetTool.Arguments == nil && targetTool.Instructions != "") {
			// The above logic will skip prompting if this is the first loop and the chat expects no args
			input, ok, err = prompter.Readline()
			if !ok || err != nil {
				return err
			}

			prog, err = prg()
			if err != nil {
				return err
			}
		}

		resp, err = chatter.Chat(ctx, prevState, prog, env, input, runner.RunOptions{})
		if err != nil {
			return err
		}
		if resp.Done {
			if chatStateSaveFile != "" {
				_ = os.Remove(chatStateSaveFile)
			}
			return nil
		}

		if resp.Content != "" {
			_, err := prompter.Printf("%s", color.RedString("< %s\n", resp.Content))
			if err != nil {
				return err
			}
		}

		if chatStateSaveFile != "" {
			_ = os.WriteFile(chatStateSaveFile, []byte(resp.Content), 0600)
		}

		prevState = resp.State
		prevResp = resp
	}
}
