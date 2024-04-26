package chat

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/adrg/xdg"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
)

var _ Prompter = (*readlinePrompter)(nil)

type readlinePrompter struct {
	readliner *readline.Instance
}

func newReadlinePrompter(prg GetProgram) (*readlinePrompter, error) {
	targetProgram, err := prg()
	if err != nil {
		return nil, err
	}

	historyFile, err := xdg.CacheFile(fmt.Sprintf("gptscript/chat-%s.history", hash.ID(targetProgram.EntryToolID)))
	if err != nil {
		historyFile = ""
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:            color.GreenString("> "),
		HistoryFile:       historyFile,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}

	l.CaptureExitSignal()
	mvl.SetOutput(l.Stderr())

	return &readlinePrompter{
		readliner: l,
	}, nil
}

func (r *readlinePrompter) Printf(format string, args ...interface{}) (int, error) {
	return fmt.Fprintf(r.readliner.Stdout(), format, args...)
}

func (r *readlinePrompter) Readline() (string, bool, error) {
	line, err := r.readliner.Readline()
	if errors.Is(err, readline.ErrInterrupt) {
		return "", false, nil
	} else if errors.Is(err, io.EOF) {
		return "", false, nil
	}
	return strings.TrimSpace(line), true, nil
}

func (r *readlinePrompter) SetPrompt(prompt string) {
	r.readliner.SetPrompt(prompt)
}

func (r *readlinePrompter) Close() error {
	return r.readliner.Close()
}
