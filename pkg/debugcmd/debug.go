package debugcmd

import (
	"context"
	"os"
	"os/exec"
)

func New(ctx context.Context, arg string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, arg, args...)
	SetupDebug(cmd)
	return cmd
}

func SetupDebug(cmd *exec.Cmd) {
	if log.IsDebug() {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
}
