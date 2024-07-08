package daemon

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func SysDaemon() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_, _ = io.ReadAll(os.Stdin)
		cancel()
	}()

	cmd := exec.CommandContext(ctx, os.Args[2], os.Args[3:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Cancel = func() error {
		if runtime.GOOS == "windows" {
			return cmd.Process.Kill()
		}
		return cmd.Process.Signal(os.Interrupt)
	}
	return cmd.Run()
}
