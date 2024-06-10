package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/google/shlex"
	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

var requiredFileExtensions = map[string]string{
	"powershell.exe": "*.ps1",
	"powershell":     "*.ps1",
}

func (e *Engine) runCommand(ctx Context, tool types.Tool, input string, toolCategory ToolCategory) (cmdOut string, cmdErr error) {
	id := counter.Next()

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
		return tool.BuiltinFunc(ctx.WrappedContext(), e.Env, input)
	}

	var instructions []string
	for _, inputContext := range ctx.InputContext {
		instructions = append(instructions, inputContext.Content)
	}
	var extraEnv = []string{
		strings.TrimSpace(fmt.Sprintf("GPTSCRIPT_CONTEXT=%s", strings.Join(instructions, "\n"))),
	}

	cmd, stop, err := e.newCommand(ctx.Ctx, extraEnv, tool, input)
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
	cmd.Stderr = io.MultiWriter(all, os.Stderr)
	cmd.Stdout = io.MultiWriter(all, output)

	if err := cmd.Run(); err != nil {
		if toolCategory == NoCategory {
			return fmt.Sprintf("ERROR: got (%v) while running tool, OUTPUT: %s", err, all), nil
		}
		_, _ = os.Stderr.Write(output.Bytes())
		log.Errorf("failed to run tool [%s] cmd %v: %v", tool.Parameters.Name, cmd.Args, err)
		return "", fmt.Errorf("ERROR: %s: %w", all, err)
	}

	return output.String(), nil
}

func (e *Engine) getRuntimeEnv(ctx context.Context, tool types.Tool, cmd, env []string) ([]string, error) {
	var (
		workdir = tool.WorkingDir
		err     error
	)
	if e.RuntimeManager != nil {
		workdir, env, err = e.RuntimeManager.GetContext(ctx, tool, cmd, env)
		if err != nil {
			return nil, err
		}
	}
	return append(env, "GPTSCRIPT_TOOL_DIR="+workdir), nil
}

func envAsMapAndDeDup(env []string) (sortedEnv []string, _ map[string]string) {
	envMap := map[string]string{}
	var keys []string
	for _, env := range env {
		key, value, _ := strings.Cut(env, "=")
		if _, existing := envMap[key]; !existing {
			keys = append(keys, key)
		}
		envMap[key] = value
	}
	sort.Strings(keys)
	for _, key := range keys {
		sortedEnv = append(sortedEnv, key+"="+envMap[key])
	}

	return sortedEnv, envMap
}

var ignoreENV = map[string]struct{}{
	"PATH":               {},
	"Path":               {},
	"GPTSCRIPT_TOOL_DIR": {},
}

func appendEnv(envs []string, k, v string) []string {
	for _, k := range []string{k, env.ToEnvLike(k)} {
		if _, ignore := ignoreENV[k]; !ignore {
			envs = append(envs, k+"="+v)
		}
	}
	return envs
}

func appendInputAsEnv(env []string, input string) []string {
	data := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()

	if err := json.Unmarshal([]byte(input), &data); err != nil {
		// ignore invalid JSON
		return env
	}

	for k, v := range data {
		switch val := v.(type) {
		case string:
			env = appendEnv(env, k, val)
		case json.Number:
			env = appendEnv(env, k, string(val))
		case bool:
			env = appendEnv(env, k, fmt.Sprint(val))
		default:
			data, err := json.Marshal(val)
			if err == nil {
				env = appendEnv(env, k, string(data))
			}
		}
	}

	env = appendEnv(env, "GPTSCRIPT_INPUT", input)
	return env
}

func (e *Engine) newCommand(ctx context.Context, extraEnv []string, tool types.Tool, input string) (*exec.Cmd, func(), error) {
	envvars := append(e.Env[:], extraEnv...)
	envvars = appendInputAsEnv(envvars, input)
	if log.IsDebug() {
		envvars = append(envvars, "GPTSCRIPT_DEBUG=true")
	}

	interpreter, rest, _ := strings.Cut(tool.Instructions, "\n")
	interpreter = strings.TrimSpace(interpreter)[2:]

	args, err := shlex.Split(interpreter)
	if err != nil {
		return nil, nil, err
	}

	envvars, err = e.getRuntimeEnv(ctx, tool, args, envvars)
	if err != nil {
		return nil, nil, err
	}

	envvars, envMap := envAsMapAndDeDup(envvars)
	for i, arg := range args {
		args[i] = os.Expand(arg, func(s string) string {
			return envMap[s]
		})
	}

	if runtime.GOOS == "windows" && (args[0] == "/usr/bin/env" || args[0] == "/bin/env") {
		args = args[1:]
	}

	var (
		cmdArgs = args[1:]
		stop    = func() {}
	)

	if strings.TrimSpace(rest) != "" {
		f, err := os.CreateTemp("", version.ProgramName+requiredFileExtensions[args[0]])
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

	// This is a workaround for Windows, where the command interpreter is constructed with unix style paths
	// It converts unix style paths to windows style paths
	if runtime.GOOS == "windows" {
		parts := strings.Split(args[0], "/")
		if parts[len(parts)-1] == "gptscript-go-tool" {
			parts[len(parts)-1] = "gptscript-go-tool.exe"
		}

		args[0] = filepath.Join(parts...)
	}

	cmd := exec.CommandContext(ctx, env.Lookup(envvars, args[0]), cmdArgs...)
	cmd.Env = envvars
	return cmd, stop, nil
}
