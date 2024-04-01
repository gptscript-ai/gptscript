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

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Ports struct {
	daemonPorts map[string]int64
	daemonLock  sync.Mutex

	startPort, endPort int64
	usedPorts          map[int64]struct{}
	daemonCtx          context.Context
	daemonClose        func()
	daemonWG           sync.WaitGroup
}

func (p *Ports) CloseDaemons() {
	p.daemonLock.Lock()
	if p.daemonCtx == nil {
		p.daemonLock.Unlock()
		return
	}
	p.daemonLock.Unlock()

	p.daemonClose()
	p.daemonWG.Wait()
}

func (p *Ports) NextPort() int64 {
	if p.startPort == 0 {
		p.startPort = 10240
		p.endPort = 11240
	}
	// This is pretty simple and inefficient approach, but also never releases ports
	count := p.endPort - p.startPort + 1
	toTry := make([]int64, 0, count)
	for i := p.startPort; i <= p.endPort; i++ {
		toTry = append(toTry, i)
	}

	rand.Shuffle(len(toTry), func(i, j int) {
		toTry[i], toTry[j] = toTry[j], toTry[i]
	})

	for _, nextPort := range toTry {
		if _, ok := p.usedPorts[nextPort]; ok {
			continue
		}
		if p.usedPorts == nil {
			p.usedPorts = map[int64]struct{}{}
		}
		p.usedPorts[nextPort] = struct{}{}
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

func (e *Engine) startDaemon(_ context.Context, tool types.Tool) (string, error) {
	e.Ports.daemonLock.Lock()
	defer e.Ports.daemonLock.Unlock()

	instructions := strings.TrimPrefix(tool.Instructions, types.DaemonPrefix)
	instructions, path := getPath(instructions)
	tool.Instructions = types.CommandPrefix + instructions

	port, ok := e.Ports.daemonPorts[tool.ID]
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	if ok {
		return url, nil
	}

	if e.Ports.daemonCtx == nil {
		e.Ports.daemonCtx, e.Ports.daemonClose = context.WithCancel(context.Background())
	}

	ctx := e.Ports.daemonCtx
	port = e.Ports.NextPort()
	url = fmt.Sprintf("http://127.0.0.1:%d%s", port, path)

	cmd, stop, err := e.newCommand(ctx, []string{
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("GPTSCRIPT_PORT=%d", port),
	},
		tool,
		"{}",
	)
	if err != nil {
		return url, err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	cmd.Stdin = r
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("launched [%s][%s] port [%d] %v", tool.Parameters.Name, tool.ID, port, cmd.Args)
	if err := cmd.Start(); err != nil {
		stop()
		return url, err
	}

	if e.Ports.daemonPorts == nil {
		e.Ports.daemonPorts = map[string]int64{}
	}
	e.Ports.daemonPorts[tool.ID] = port

	killedCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("daemon exited tool [%s] %v: %v", tool.Parameters.Name, cmd.Args, err)
		}
		_ = r.Close()
		_ = w.Close()

		cancel(err)
		stop()
		e.Ports.daemonLock.Lock()
		defer e.Ports.daemonLock.Unlock()

		delete(e.Ports.daemonPorts, tool.ID)
	}()

	e.Ports.daemonWG.Add(1)
	context.AfterFunc(ctx, func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Debugf("daemon failed to kill tool [%s] process: %v", tool.Parameters.Name, err)
		}
		e.Ports.daemonWG.Done()
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
