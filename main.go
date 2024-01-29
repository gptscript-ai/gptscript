package main

import (
	"github.com/acorn-io/cmd"
	"github.com/acorn-io/gptscript/pkg/cli"
)

func main() {
	cmd.Main(cli.New())
}
