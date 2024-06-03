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
		Version: "21",
		Default: true,
	},
	&golang.Runtime{
		Version: "1.22.1",
	},
}

func Default(cacheDir string) engine.RuntimeManager {
	return repos.New(cacheDir, Runtimes...)
}
