package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	context2 "github.com/gptscript-ai/gptscript/pkg/context"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

func sysPromptHTTP(ctx context.Context, envs []string, url string, prompt types.Prompt) (_ string, err error) {
	data, err := json.Marshal(prompt)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	for _, env := range envs {
		if _, v, ok := strings.Cut(env, types.PromptTokenEnvVar+"="); ok && v != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v))
			break
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("getting prompt response invalid status code [%d], expected 200", resp.StatusCode)
	}

	data, err = io.ReadAll(resp.Body)
	return string(data), err
}

func SysPrompt(ctx context.Context, envs []string, input string, _ chan<- string) (_ string, err error) {
	var params struct {
		Message   string `json:"message,omitempty"`
		Fields    string `json:"fields,omitempty"`
		Sensitive string `json:"sensitive,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", err
	}

	for _, env := range envs {
		if url, ok := strings.CutPrefix(env, types.PromptURLEnvVar+"="); ok {
			httpPrompt := types.Prompt{
				Message:   params.Message,
				Fields:    strings.Split(params.Fields, ","),
				Sensitive: params.Sensitive == "true",
			}
			return sysPromptHTTP(ctx, envs, url, httpPrompt)
		}
	}

	return "", fmt.Errorf("no prompt server found, can not continue")
}

func sysPrompt(ctx context.Context, req types.Prompt) (_ string, err error) {
	defer context2.GetPauseFuncFromCtx(ctx)()()

	if req.Message != "" && len(req.Fields) == 1 && strings.TrimSpace(req.Fields[0]) == "" {
		var errs []error
		_, err := fmt.Fprintln(os.Stderr, req.Message)
		errs = append(errs, err)
		_, err = fmt.Fprintln(os.Stderr, "Press enter to continue...")
		errs = append(errs, err)
		_, err = fmt.Fscanln(os.Stdin)
		errs = append(errs, err)
		return "", errors.Join(errs...)
	}

	if req.Message != "" && len(req.Fields) != 1 {
		_, _ = fmt.Fprintln(os.Stderr, req.Message)
	}

	results := map[string]string{}
	for _, f := range req.Fields {
		var (
			value string
			msg   = f
		)
		if len(req.Fields) == 1 && req.Message != "" {
			msg = req.Message
		}
		if req.Sensitive {
			err = survey.AskOne(&survey.Password{Message: msg}, &value, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		} else {
			err = survey.AskOne(&survey.Input{Message: msg}, &value, survey.WithStdio(os.Stdin, os.Stderr, os.Stderr))
		}
		if err != nil {
			return "", err
		}
		results[f] = value
	}

	resultsStr, err := json.Marshal(results)
	if err != nil {
		return "", err
	}

	return string(resultsStr), nil
}
