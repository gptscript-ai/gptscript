package main

import (
	"github.com/acorn-io/cmd"
	"github.com/gptscript-ai/gptscript/pkg/cli"
)

func main() {
	cmd.Main(cli.New())
}
