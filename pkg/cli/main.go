package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/gptscript-ai/cmd"
	"github.com/gptscript-ai/gptscript/pkg/daemon"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/nanobot/supervise"
)

func Main() {
	if len(os.Args) > 2 {
		if os.Args[1] == "sys.daemon" {
			if os.Getenv("GPTSCRIPT_DEBUG") == "true" {
				mvl.SetDebug()
			}
			if err := daemon.SysDaemon(); err != nil {
				log.Debugf("failed running daemon: %v", err)
			}
			os.Exit(0)
		}
		if os.Args[1] == "_exec" {
			if err := supervise.Daemon(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "failed running _exec: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cmd.MainCtx(ctx, New())
}
