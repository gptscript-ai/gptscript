package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

const DaemonURLSuffix = ".daemon.gptscript.local"

func (e *Engine) runHTTP(ctx Context, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	envMap := map[string]string{}

	for _, env := range appendInputAsEnv(nil, input) {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	toolURL := strings.Split(tool.Instructions, "\n")[0][2:]
	toolURL = os.Expand(toolURL, func(s string) string {
		return url.PathEscape(envMap[s])
	})

	parsed, err := url.Parse(toolURL)
	if err != nil {
		return nil, err
	}

	var (
		requestedEnvVars map[string]struct{}
		daemonToken      string
	)
	if strings.HasSuffix(parsed.Hostname(), DaemonURLSuffix) {
		referencedToolName := strings.TrimSuffix(parsed.Hostname(), DaemonURLSuffix)
		referencedToolRefs, ok := tool.ToolMapping[referencedToolName]
		if !ok || len(referencedToolRefs) != 1 {
			return nil, fmt.Errorf("invalid reference [%s] to tool [%s] from [%s], missing \"tools: %s\" parameter", toolURL, referencedToolName, tool.Source, referencedToolName)
		}
		referencedTool, ok := ctx.Program.ToolSet[referencedToolRefs[0].ToolID]
		if !ok {
			return nil, fmt.Errorf("failed to find tool [%s] for [%s]", referencedToolName, parsed.Hostname())
		}
		toolURL, daemonToken, err = e.startDaemon(referencedTool)
		if err != nil {
			return nil, err
		}
		toolURLParsed, err := url.Parse(toolURL)
		if err != nil {
			return nil, err
		}
		parsed.Host = toolURLParsed.Host
		toolURL = parsed.String()

		metadataEnvVars := strings.Split(referencedTool.MetaData["requestedEnvVars"], ",")
		requestedEnvVars = make(map[string]struct{}, len(metadataEnvVars))
		for _, e := range metadataEnvVars {
			if e != "" {
				requestedEnvVars[e] = struct{}{}
			}
		}
	}

	if tool.Blocking {
		return &Return{
			Result: &toolURL,
		}, nil
	}

	if body, ok := envMap["BODY"]; ok {
		input = body
	}

	req, err := http.NewRequestWithContext(ctx.Ctx, http.MethodPost, toolURL, strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	if daemonToken != "" {
		req.Header.Add("X-GPTScript-Daemon-Token", daemonToken)
	}

	for _, k := range slices.Sorted(maps.Keys(envMap)) {
		if _, ok := requestedEnvVars[k]; ok || strings.HasPrefix(k, "GPTSCRIPT_WORKSPACE_") {
			req.Header.Add("X-GPTScript-Env", k+"="+envMap[k])
		}
	}

	for _, prefix := range strings.Split(envMap["GPTSCRIPT_HTTP_ENV_PREFIX"], ",") {
		if prefix == "" {
			continue
		}
		for _, k := range slices.Sorted(maps.Keys(envMap)) {
			if strings.HasPrefix(k, prefix) {
				req.Header.Add("X-GPTScript-Env", k+"="+envMap[k])
			}
		}
	}

	for _, k := range strings.Split(envMap["GPTSCRIPT_HTTP_ENV"], ",") {
		if k == "" {
			continue
		}
		v := envMap[k]
		if v != "" {
			req.Header.Add("X-GPTScript-Env", k+"="+v)
		}
	}

	req.Header.Set("X-GPTScript-Tool-Name", tool.Name)

	if err := json.Unmarshal([]byte(input), &map[string]any{}); err == nil {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "text/plain")
	}

	// If the user canceled the run, then don't make the request.
	select {
	case <-ctx.userCancel:
		return &Return{}, nil
	default:
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error in request to [%s] [%d]: %s: %s", toolURL, resp.StatusCode, resp.Status, body)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Type") == "application/json" && strings.HasPrefix(string(content), "\"") {
		// This is dumb hack when something returns a string in JSON format, just decode it to a string
		var s string
		if err := json.Unmarshal(content, &s); err == nil {
			return &Return{
				Result: &s,
			}, nil
		}
	}

	s := string(content)
	return &Return{
		Result: &s,
	}, nil
}
