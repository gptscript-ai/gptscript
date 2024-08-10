package engine

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gptscript-ai/gptscript/pkg/system"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

var ports Ports

type Ports struct {
	daemonPorts map[string]int64
	daemonLock  sync.Mutex

	startPort, endPort int64
	usedPorts          map[int64]struct{}
	daemonCtx          context.Context
	daemonClose        func()
	daemonWG           sync.WaitGroup
}

func SetPorts(start, end int64) {
	ports.daemonLock.Lock()
	defer ports.daemonLock.Unlock()
	if ports.startPort == 0 && ports.endPort == 0 {
		ports.startPort = start
		ports.endPort = end
	}
}

func CloseDaemons() {
	ports.daemonLock.Lock()
	if ports.daemonCtx == nil {
		ports.daemonLock.Unlock()
		return
	}
	ports.daemonLock.Unlock()

	ports.daemonClose()
	ports.daemonWG.Wait()
}

func nextPort() int64 {
	if ports.startPort == 0 {
		ports.startPort = 10240
		ports.endPort = 11240
	}
	// This is pretty simple and inefficient approach, but also never releases ports
	count := ports.endPort - ports.startPort + 1
	toTry := make([]int64, 0, count)
	for i := ports.startPort; i <= ports.endPort; i++ {
		toTry = append(toTry, i)
	}

	rand.Shuffle(len(toTry), func(i, j int) {
		toTry[i], toTry[j] = toTry[j], toTry[i]
	})

	for _, nextPort := range toTry {
		if _, ok := ports.usedPorts[nextPort]; ok {
			continue
		}
		if ports.usedPorts == nil {
			ports.usedPorts = map[int64]struct{}{}
		}
		ports.usedPorts[nextPort] = struct{}{}
		return nextPort
	}

	panic("Ran out of usable ports")
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

func (e *Engine) startDaemon(tool types.Tool) (string, error) {
	ports.daemonLock.Lock()
	defer ports.daemonLock.Unlock()

	instructions := strings.TrimPrefix(tool.Instructions, types.DaemonPrefix)
	instructions, path := getPath(instructions)
	tool.Instructions = types.CommandPrefix + instructions

	port, ok := ports.daemonPorts[tool.ID]
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	if ok {
		return url, nil
	}

	if ports.daemonCtx == nil {
		var cancel func()
		ports.daemonCtx, cancel = context.WithCancel(context.Background())
		ports.daemonClose = func() {
			cancel()
			ports.daemonCtx = nil
		}
	}

	ctx := ports.daemonCtx
	port = nextPort()
	url = fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	cmd, stop, err := e.newCommand(ctx, []string{
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("GPTSCRIPT_PORT=%d", port),
	},
		tool,
		"{}",
		false,
	)
	if err != nil {
		return url, err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// Loop back to gptscript to help with process supervision
	cmd.Args = append([]string{system.Bin(), "sys.daemon", cmd.Path}, cmd.Args[1:]...)
	cmd.Path = system.Bin()

	cmd.Stdin = r
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Cancel = func() error {
		_ = r.Close()
		return w.Close()
	}

	log.Infof("launched [%s][%s] port [%d] %v", tool.Parameters.Name, tool.ID, port, cmd.Args)
	if err := cmd.Start(); err != nil {
		stop()
		return url, err
	}

	if ports.daemonPorts == nil {
		ports.daemonPorts = map[string]int64{}
	}
	ports.daemonPorts[tool.ID] = port

	killedCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	ports.daemonWG.Add(1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Debugf("daemon exited tool [%s] %v: %v", tool.Parameters.Name, cmd.Args, err)
		}
		_ = r.Close()
		_ = w.Close()

		cancel(err)
		stop()
		ports.daemonLock.Lock()
		defer ports.daemonLock.Unlock()

		delete(ports.daemonPorts, tool.ID)
		ports.daemonWG.Done()
	}()

	for i := 0; i < 120; i++ {
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
	url, err := e.startDaemon(tool)
	if err != nil {
		return nil, err
	}

	tool.Instructions = strings.Join(append([]string{
		types.CommandPrefix + url,
	}, strings.Split(tool.Instructions, "\n")[1:]...), "\n")
	return e.runHTTP(ctx, prg, tool, input)
}
