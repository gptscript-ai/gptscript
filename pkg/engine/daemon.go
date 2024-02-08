package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) getNextPort() int64 {
	count := e.endPort - e.startPort
	e.nextPort++
	e.nextPort = e.nextPort % count
	return e.startPort + e.nextPort
}

func (e *Engine) startDaemon(ctx context.Context, tool types.Tool) (string, error) {
	e.daemonLock.Lock()
	defer e.daemonLock.Unlock()

	port, ok := e.daemonPorts[tool.ID]
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	if ok {
		return url, nil
	}

	port = e.getNextPort()

	instructions := strings.TrimPrefix(tool.Instructions, "#!daemon ")
	cmd, close, err := e.newCommand(ctx, []string{
		fmt.Sprintf("PORT=%d", port),
	},
		instructions,
		"{}",
	)
	if err != nil {
		return url, err
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("launching   [%s] port [%d] %v", tool.Name, port, cmd.Args)
	if err := cmd.Start(); err != nil {
		close()
		return url, err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Errorf("daemon exited tool [%s] %v: %v", tool.Name, cmd.Args, err)
		}

		close()
		e.daemonLock.Lock()
		defer e.daemonLock.Unlock()

		delete(e.daemonPorts, tool.ID)
	}()

	context.AfterFunc(ctx, func() {
		cmd.Process.Kill()
	})

	for range 20 {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer func() {
				_ = resp.Body.Close()
			}()
			go func() {
				_, _ = io.ReadAll(resp.Body)
			}()
			return url, nil
		}
		select {
		case <-ctx.Done():
			return url, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return url, fmt.Errorf("timeout waiting for 200 response from GET %s", url)
}

func (e *Engine) runDaemon(ctx context.Context, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	url, err := e.startDaemon(ctx, tool)
	if err != nil {
		return nil, err
	}

	tool.Instructions = strings.Join(append([]string{url},
		strings.Split(tool.Instructions, "\n")[1:]...), "\n")
	return e.runHTTP(ctx, tool, input)
}
