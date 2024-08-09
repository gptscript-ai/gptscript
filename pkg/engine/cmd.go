package engine

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/anmitsu/go-shlex"
	"github.com/gptscript-ai/gptscript/pkg/counter"
	"github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

var requiredFileExtensions = map[string]string{
	"powershell.exe": "*.ps1",
	"powershell":     "*.ps1",
}

type outputWriter struct {
	id       string
	progress chan<- types.CompletionStatus
	buf      bytes.Buffer
}

func (o *outputWriter) Write(p []byte) (n int, err error) {
	o.buf.Write(p)
	o.progress <- types.CompletionStatus{
		CompletionID: o.id,
		PartialResponse: &types.CompletionMessage{
			Role:    types.CompletionMessageRoleTypeAssistant,
			Content: types.Text(o.buf.String()),
		},
	}
	return len(p), nil
}

func compressEnv(envs []string) (result []string) {
	for _, env := range envs {
		k, v, ok := strings.Cut(env, "=")
		if !ok || len(v) < 40_000 {
			result = append(result, env)
			continue
		}

		out := bytes.NewBuffer(nil)
		b64 := base64.NewEncoder(base64.StdEncoding, out)
		gz := gzip.NewWriter(b64)
		_, _ = gz.Write([]byte(v))
		_ = gz.Close()
		_ = b64.Close()
		result = append(result, k+`={"_gz":"`+out.String()+`"}`)
	}
	return
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

		var (
			progress = make(chan string)
			wg       sync.WaitGroup
		)
		wg.Add(1)
		defer wg.Wait()
		defer close(progress)
		go func() {
			defer wg.Done()
			buf := strings.Builder{}
			for line := range progress {
				buf.WriteString(line)
				e.Progress <- types.CompletionStatus{
					CompletionID: id,
					PartialResponse: &types.CompletionMessage{
						Role:    types.CompletionMessageRoleTypeAssistant,
						Content: types.Text(buf.String()),
					},
				}
			}
		}()

		return tool.BuiltinFunc(ctx.WrappedContext(), e.Env, input, progress)
	}

	var instructions []string
	for _, inputContext := range ctx.InputContext {
		instructions = append(instructions, inputContext.Content)
	}

	var extraEnv = []string{
		strings.TrimSpace("GPTSCRIPT_CONTEXT=" + strings.Join(instructions, "\n")),
	}
	cmd, stop, err := e.newCommand(ctx.Ctx, extraEnv, tool, input)
	if err != nil {
		if toolCategory == NoCategory {
			return fmt.Sprintf("ERROR: got (%v) while parsing command", err), nil
		}
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

	var (
		stdout       = &bytes.Buffer{}
		stdoutAndErr = &bytes.Buffer{}
		progressOut  = &outputWriter{
			id:       id,
			progress: e.Progress,
		}
		result *bytes.Buffer
	)

	cmd.Stdout = io.MultiWriter(stdout, stdoutAndErr, progressOut)
	if toolCategory == NoCategory || toolCategory == ContextToolCategory {
		cmd.Stderr = io.MultiWriter(stdoutAndErr, progressOut)
		result = stdoutAndErr
	} else {
		cmd.Stderr = io.MultiWriter(stdoutAndErr, progressOut, os.Stderr)
		result = stdout
	}

	if err := cmd.Run(); err != nil {
		if toolCategory == NoCategory {
			return fmt.Sprintf("ERROR: got (%v) while running tool, OUTPUT: %s", err, stdoutAndErr), nil
		}
		log.Errorf("failed to run tool [%s] cmd %v: %v", tool.Parameters.Name, cmd.Args, err)
		return "", fmt.Errorf("ERROR: %s: %w", result, err)
	}

	return result.String(), IsChatFinishMessage(result.String())
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
	if _, ignore := ignoreENV[k]; !ignore {
		envs = append(envs, strings.ToUpper(env.ToEnvLike(k))+"="+v)
	}
	return envs
}

func appendInputAsEnv(env []string, input string) []string {
	data := map[string]any{}
	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()

	env = appendEnv(env, "GPTSCRIPT_INPUT", input)

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

	args, err := shlex.Split(interpreter, runtime.GOOS != "windows")
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

	// After we determined the interpreter we again interpret the args by env vars
	args, err = replaceVariablesForInterpreter(interpreter, envMap)
	if err != nil {
		return nil, nil, err
	}

	if runtime.GOOS == "windows" && (args[0] == "/bin/bash" || args[0] == "/bin/sh") {
		args[0] = path.Base(args[0])
	}

	if runtime.GOOS == "windows" && (args[0] == "/usr/bin/env" || args[0] == "/bin/env") {
		args = args[1:]
	}

	var (
		cmdArgs = args[1:]
		stop    = func() {}
	)

	if strings.TrimSpace(rest) != "" {
		f, err := os.CreateTemp(env.Getenv("GPTSCRIPT_TMPDIR", envvars), version.ProgramName+requiredFileExtensions[args[0]])
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
	cmd.Env = compressEnv(envvars)
	return cmd, stop, nil
}

func replaceVariablesForInterpreter(interpreter string, envMap map[string]string) ([]string, error) {
	var parts []string
	for i, part := range splitByQuotes(interpreter) {
		if i%2 == 0 {
			part = os.Expand(part, func(s string) string {
				return envMap[s]
			})
			// We protect newly resolved env vars from getting replaced when we do the second Expand
			// after shlex. Yeah, crazy. I'm guessing this isn't secure, but just trying to avoid a foot gun.
			part = os.Expand(part, func(s string) string {
				return "${__" + s + "}"
			})
		}
		parts = append(parts, part)
	}

	parts, err := shlex.Split(strings.Join(parts, ""), runtime.GOOS != "windows")
	if err != nil {
		return nil, err
	}

	for i, part := range parts {
		parts[i] = os.Expand(part, func(s string) string {
			if strings.HasPrefix(s, "__") {
				return "${" + s[2:] + "}"
			}
			return envMap[s]
		})
	}

	return parts, nil
}

// splitByQuotes will split a string by parsing matching double quotes (with \ as the escape character).
// The return value conforms to the following properties
//  1. s == strings.Join(result, "")
//  2. Even indexes are strings that were not in quotes.
//  3. Odd indexes are strings that were quoted.
//
// Example: s = `In a "quoted string" quotes can be escaped with \"`
//
//	result = [`In a `, `"quoted string"`, ` quotes can be escaped with \"`]
func splitByQuotes(s string) (result []string) {
	var (
		buf               strings.Builder
		inEscape, inQuote bool
	)

	for _, c := range s {
		if inEscape {
			buf.WriteRune(c)
			inEscape = false
			continue
		}

		switch c {
		case '"':
			if inQuote {
				buf.WriteRune(c)
			}
			result = append(result, buf.String())
			buf.Reset()
			if !inQuote {
				buf.WriteRune(c)
			}
			inQuote = !inQuote
		case '\\':
			inEscape = true
			buf.WriteRune(c)
		default:
			buf.WriteRune(c)
		}
	}

	if buf.Len() > 0 {
		if inQuote {
			result = append(result, "")
		}
		result = append(result, buf.String())
	}

	return
}
