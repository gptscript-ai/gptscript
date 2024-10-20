package cli

import (
	"context"
	"os"

	"github.com/gptscript-ai/gptscript/pkg/sdkserver"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type SDKServer struct {
	*GPTScript
	WorkspaceTool string `usage:"Tool to use for workspace"`
}

func (c *SDKServer) Customize(cmd *cobra.Command) {
	cmd.Use = "sys.sdkserver"
	cmd.Args = cobra.NoArgs
	cmd.Aliases = []string{"sdkserver"}
	cmd.Hidden = true
}

func (c *SDKServer) Run(cmd *cobra.Command, _ []string) error {
	opts, err := c.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	// Don't use cmd.Context() as we don't want to die on ctrl+c
	ctx := context.Background()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Only support CTRL+C if stdin is the terminal. When ran as an SDK it will be a pipe
		ctx = cmd.Context()
	}

	return sdkserver.Run(ctx, sdkserver.Options{
		Options:       opts,
		ListenAddress: c.ListenAddress,
		Debug:         c.Debug,
		WorkspaceTool: c.WorkspaceTool,
	})
}
