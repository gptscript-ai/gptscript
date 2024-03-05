package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/google/shlex"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

var (
	reservedEnvNameList = map[string]struct{}{
		"PATH": {},
	}
)

func (e *Engine) runCommand(ctx context.Context, tool types.Tool, input string) (cmdOut string, cmdErr error) {
	id := fmt.Sprint(atomic.AddInt64(&completionID, 1))

	defer func() {
		e.Progress <- types.CompletionStatus{
			CompletionID: id,
			Response: map[string]any{
				"output": cmdOut,
				"err":    cmdErr,
			},
		}
	}()

	if tool.BuiltinFunc != nil {
		e.Progress <- types.CompletionStatus{
			CompletionID: id,
			Request: map[string]any{
				"command": []string{tool.ID},
				"input":   input,
			},
		}
		return tool.BuiltinFunc(ctx, e.Env, input)
	}

	var extraEnv []string
	if tool.WorkingDir != "" {
		extraEnv = append(extraEnv, "GPTSCRIPT_TOOL_DIR="+tool.WorkingDir)
	}

	cmd, stop, err := e.newCommand(ctx, extraEnv, tool.Instructions, input)
	if err != nil {
		return "", err
	}
	defer stop()

	e.Progress <- types.CompletionStatus{
		CompletionID: id,
		Request: map[string]any{
			"command": cmd.Args,
			"input":   input,
		},
	}

	output := &bytes.Buffer{}
	all := &bytes.Buffer{}
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = io.MultiWriter(all, os.Stderr)
	cmd.Stdout = io.MultiWriter(all, output)

	if err := cmd.Run(); err != nil {
		_, _ = os.Stderr.Write(output.Bytes())
		log.Errorf("failed to run tool [%s] cmd %v: %v", tool.Parameters.Name, cmd.Args, err)
		return "", fmt.Errorf("ERROR: %s: %w", all, err)
	}

	return output.String(), nil
}

func (e *Engine) newCommand(ctx context.Context, extraEnv []string, instructions, input string) (*exec.Cmd, func(), error) {
	env := append(e.Env[:], extraEnv...)
	data := map[string]any{}

	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()

	envMap := map[string]string{}
	for _, env := range env {
		key, value, _ := strings.Cut(env, "=")
		envMap[key] = value
	}

	if err := json.Unmarshal([]byte(input), &data); err == nil {
		for k, v := range data {
			envName := strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
			switch val := v.(type) {
			case string:
				envMap[envName] = val
				env = appendEnvName(env, envName, val)
				envMap[k] = val
				env = append(env, k+"="+val)
			case json.Number:
				envMap[envName] = string(val)
				env = appendEnvName(env, envName, string(val))
				envMap[k] = string(val)
				env = append(env, k+"="+string(val))
			case bool:
				envMap[envName] = fmt.Sprint(val)
				env = appendEnvName(env, envName, fmt.Sprint(val))
				envMap[k] = fmt.Sprint(val)
				env = append(env, k+"="+fmt.Sprint(val))
			default:
				data, err := json.Marshal(val)
				if err == nil {
					envMap[envName] = string(data)
					env = appendEnvName(env, envName, string(data))
					envMap[k] = string(data)
					env = append(env, k+"="+string(data))
				}
			}
		}
	}

	interpreter, rest, _ := strings.Cut(instructions, "\n")
	interpreter = strings.TrimSpace(interpreter)[2:]

	interpreter = os.Expand(interpreter, func(s string) string {
		return envMap[s]
	})

	args, err := shlex.Split(interpreter)
	if err != nil {
		return nil, nil, err
	}

	var (
		cmdArgs = args[1:]
		stop    = func() {}
	)

	if strings.TrimSpace(rest) != "" {
		f, err := os.CreateTemp("", version.ProgramName)
		if err != nil {
			return nil, nil, err
		}
		stop = func() {
			_ = os.Remove(f.Name())
		}

		_, err = f.Write([]byte(rest))
		_ = f.Close()
		if err != nil {
			stop()
			return nil, nil, err
		}
		cmdArgs = append(cmdArgs, f.Name())
	}

	cmd := exec.CommandContext(ctx, args[0], cmdArgs...)
	cmd.Env = env
	return cmd, stop, nil
}

// appendEnvName appends the name and value to the env slice if the name is not in the reservedEnvNameList
func appendEnvName(env []string, name, value string) []string {
	if _, ok := reservedEnvNameList[name]; ok {
		return env
	}
	return append(env, name+"="+value)
}
