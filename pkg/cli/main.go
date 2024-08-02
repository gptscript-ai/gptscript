package cli

import (
	"context"
	"os"
	"os/signal"

	"github.com/gptscript-ai/cmd"
	"github.com/gptscript-ai/gptscript/pkg/daemon"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
)

func Main() {
	if len(os.Args) > 2 && os.Args[1] == "sys.daemon" {
		if os.Getenv("GPTSCRIPT_DEBUG") == "true" {
			mvl.SetDebug()
		}
		if err := daemon.SysDaemon(); err != nil {
			log.Debugf("failed running daemon: %v", err)
		}
		os.Exit(0)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cmd.MainCtx(ctx, New())
}
