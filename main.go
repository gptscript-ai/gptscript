package main

import (
	"github.com/gptscript-ai/gptscript/pkg/cli"
	// Load all VCS
	_ "github.com/gptscript-ai/gptscript/pkg/loader/vcs"
)

func main() {
	cli.Main()
}
