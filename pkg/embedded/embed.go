package embedded

import (
	"io/fs"
	"os"
	"strings"

	"github.com/gptscript-ai/gptscript/internal"
	"github.com/gptscript-ai/gptscript/pkg/cli"
	"github.com/gptscript-ai/gptscript/pkg/system"
)

type Options struct {
	FS fs.FS
}

func Run(opts ...Options) bool {
	for _, opt := range opts {
		if opt.FS == nil {
			internal.FS = opt.FS
		}
	}

	system.SetBinToSelf()
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "sys.") {
		cli.Main()
		return true
	}

	return false
}
