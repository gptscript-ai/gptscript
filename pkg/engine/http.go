package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

const DaemonURLSuffix = ".daemon.gpt.local"

func (e *Engine) runHTTP(ctx context.Context, prg *types.Program, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	envMap := map[string]string{}

	for _, env := range e.Env {
		k, v, _ := strings.Cut(env, "=")
		envMap[k] = v
	}

	toolURL := strings.Split(tool.Instructions, "\n")[0][2:]
	toolURL = os.Expand(toolURL, func(s string) string {
		return envMap[s]
	})

	parsed, err := url.Parse(toolURL)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(parsed.Hostname(), DaemonURLSuffix) {
		referencedToolName := strings.TrimSuffix(parsed.Hostname(), DaemonURLSuffix)
		referencedToolID, ok := tool.ToolMapping[referencedToolName]
		if !ok {
			return nil, fmt.Errorf("invalid reference [%s] to tool [%s] from [%s], missing \"tools: %s\" parameter", toolURL, referencedToolName, tool.Source, referencedToolName)
		}
		referencedTool, ok := prg.ToolSet[referencedToolID]
		if !ok {
			return nil, fmt.Errorf("failed to find tool [%s] for [%s]", referencedToolName, parsed.Hostname())
		}
		toolURL, err = e.startDaemon(ctx, referencedTool)
		if err != nil {
			return nil, err
		}
		toolURLParsed, err := url.Parse(toolURL)
		if err != nil {
			return nil, err
		}
		parsed.Host = toolURLParsed.Host
		toolURL = parsed.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, toolURL, strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GPTScript-Tool-Name", tool.Parameters.Name)

	if err := json.Unmarshal([]byte(input), &map[string]any{}); err == nil {
		req.Header.Set("Content-Type", "application/json")
	} else {
		req.Header.Set("Content-Type", "text/plain")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		_, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error in request to [%s] [%d]: %s", toolURL, resp.StatusCode, resp.Status)
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
