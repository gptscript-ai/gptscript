package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runHTTP(ctx context.Context, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	url := strings.Split(tool.Instructions, "\n")[0][2:]

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GPTScript-Tool-Name", tool.Name)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		_, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error in request to [%s] [%d]: %s", url, resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	s := string(content)
	return &Return{
		Result: &s,
	}, nil
}
