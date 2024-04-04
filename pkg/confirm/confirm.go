package confirm

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

type Confirm interface {
	Confirm(ctx context.Context, prompt string) error
}

type confirmer struct{}

func WithConfirm(ctx context.Context, c Confirm) context.Context {
	return context.WithValue(ctx, confirmer{}, c)
}

func Promptf(ctx context.Context, fmtString string, args ...any) error {
	c, ok := ctx.Value(confirmer{}).(Confirm)
	if !ok {
		return nil
	}
	return c.Confirm(ctx, fmt.Sprintf(fmtString, args...))
}

type TextPrompt struct {
}

func (t TextPrompt) Confirm(_ context.Context, prompt string) error {
	var result bool
	err := survey.AskOne(&survey.Confirm{
		Message: prompt,
		Default: false,
	}, &result)
	if err != nil {
		return err
	}
	if !result {
		return errors.New("abort")
	}
	return nil
}
