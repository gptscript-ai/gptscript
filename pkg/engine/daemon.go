package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

var (
	daemonPorts map[string]int64
	daemonLock  sync.Mutex

	startPort, endPort int64
	nextPort           int64
)

func (e *Engine) getNextPort() int64 {
	if startPort == 0 {
		startPort = 10240
		endPort = 11240
	}
	count := endPort - startPort
	nextPort++
	nextPort = nextPort % count
	return startPort + nextPort
}

func (e *Engine) startDaemon(ctx context.Context, tool types.Tool) (string, error) {
	daemonLock.Lock()
	defer daemonLock.Unlock()

	port, ok := daemonPorts[tool.ID]
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	if ok {
		return url, nil
	}

	port = e.getNextPort()
	url = fmt.Sprintf("http://127.0.0.1:%d", port)

	instructions := types.CommandPrefix + strings.TrimPrefix(tool.Instructions, types.DaemonPrefix)
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
	log.Infof("launched [%s] port [%d] %v", tool.Name, port, cmd.Args)
	if err := cmd.Start(); err != nil {
		close()
		return url, err
	}

	if daemonPorts == nil {
		daemonPorts = map[string]int64{}
	}

	killedCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("daemon exited tool [%s] %v: %v", tool.Name, cmd.Args, err)
		}

		cancel(err)
		close()
		daemonLock.Lock()
		defer daemonLock.Unlock()

		delete(daemonPorts, tool.ID)
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
		case <-killedCtx.Done():
			return url, fmt.Errorf("daemon failed to start: %w", context.Cause(killedCtx))
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

	tool.Instructions = strings.Join(append([]string{
		types.CommandPrefix + url,
	}, strings.Split(tool.Instructions, "\n")[1:]...), "\n")
	return e.runHTTP(ctx, tool, input)
}
