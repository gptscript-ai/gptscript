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
	daemonCtx          context.Context
	daemonClose        func()
	daemonWG           sync.WaitGroup
)

func CloseDaemons() {
	daemonLock.Lock()
	if daemonCtx == nil {
		daemonLock.Unlock()
		return
	}
	daemonLock.Unlock()

	daemonClose()
	daemonWG.Wait()
}

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

func getPath(instructions string) (string, string) {
	instructions = strings.TrimSpace(instructions)
	line := strings.TrimSpace(instructions)

	if !strings.HasPrefix(line, "(") {
		return instructions, ""
	}

	line, rest, ok := strings.Cut(line[1:], ")")
	if !ok {
		return instructions, ""
	}

	path, value, ok := strings.Cut(strings.TrimSpace(line), "=")
	if !ok || strings.TrimSpace(path) != "path" {
		return instructions, ""
	}

	return strings.TrimSpace(rest), strings.TrimSpace(value)
}

func (e *Engine) startDaemon(_ context.Context, tool types.Tool) (string, error) {
	daemonLock.Lock()
	defer daemonLock.Unlock()

	instructions := strings.TrimPrefix(tool.Instructions, types.DaemonPrefix)
	instructions, path := getPath(instructions)

	port, ok := daemonPorts[tool.ID]
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	if ok {
		return url, nil
	}

	if daemonCtx == nil {
		daemonCtx, daemonClose = context.WithCancel(context.Background())
	}

	ctx := daemonCtx
	port = e.getNextPort()
	url = fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	cmd, stop, err := e.newCommand(ctx, []string{
		fmt.Sprintf("PORT=%d", port),
	},
		types.CommandPrefix+instructions,
		"{}",
	)
	if err != nil {
		return url, err
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("launched [%s][%s] port [%d] %v", tool.Parameters.Name, tool.ID, port, cmd.Args)
	if err := cmd.Start(); err != nil {
		stop()
		return url, err
	}

	if daemonPorts == nil {
		daemonPorts = map[string]int64{}
	}
	daemonPorts[tool.ID] = port

	killedCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("daemon exited tool [%s] %v: %v", tool.Parameters.Name, cmd.Args, err)
		}

		cancel(err)
		stop()
		daemonLock.Lock()
		defer daemonLock.Unlock()

		delete(daemonPorts, tool.ID)
	}()

	daemonWG.Add(1)
	context.AfterFunc(ctx, func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("daemon failed to kill tool [%s] process: %v", tool.Parameters.Name, err)
		}
		daemonWG.Done()
	})

	for i := 0; i < 20; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			go func() {
				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
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

func (e *Engine) runDaemon(ctx context.Context, prg *types.Program, tool types.Tool, input string) (cmdRet *Return, cmdErr error) {
	url, err := e.startDaemon(ctx, tool)
	if err != nil {
		return nil, err
	}

	tool.Instructions = strings.Join(append([]string{
		types.CommandPrefix + url,
	}, strings.Split(tool.Instructions, "\n")[1:]...), "\n")
	return e.runHTTP(ctx, prg, tool, input)
}
