package main

import (
	"os"

	"github.com/acorn-io/cmd"
	"github.com/gptscript-ai/gptscript/pkg/cli"
	"github.com/gptscript-ai/gptscript/pkg/daemon"
	"github.com/gptscript-ai/gptscript/pkg/mvl"

	// Load all VCS
	_ "github.com/gptscript-ai/gptscript/pkg/loader/vcs"
)

var log = mvl.Package()

func main() {
	if len(os.Args) > 2 && os.Args[1] == "sys.daemon" {
		if os.Getenv("GPTSCRIPT_DEBUG") == "true" {
			mvl.SetDebug()
		}
		if err := daemon.SysDaemon(); err != nil {
			log.Debugf("failed running daemon: %v", err)
		}
		os.Exit(0)
	}
	cmd.Main(cli.New())
}
