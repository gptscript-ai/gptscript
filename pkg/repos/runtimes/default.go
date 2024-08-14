package runtimes

import (
	"github.com/gptscript-ai/gptscript/pkg/engine"
	"github.com/gptscript-ai/gptscript/pkg/repos"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/busybox"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/golang"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/node"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/python"
)

var Runtimes = []repos.Runtime{
	&busybox.Runtime{},
	&python.Runtime{
		Version: "3.12",
		Default: true,
	},
	&python.Runtime{
		Version: "3.11",
	},
	&python.Runtime{
		Version: "3.10",
	},
	&node.Runtime{
		Version: "20",
		Default: true,
	},
	&golang.Runtime{
		Version: "1.23.0",
	},
}

func Default(cacheDir string) engine.RuntimeManager {
	return repos.New(cacheDir, Runtimes...)
}
