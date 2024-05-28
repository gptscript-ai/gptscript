package cli

import (
	"github.com/gptscript-ai/gptscript/pkg/sdkserver"
	"github.com/spf13/cobra"
)

type SDKServer struct {
	*GPTScript
}

func (c *SDKServer) Customize(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
	cmd.Hidden = true
}

func (c *SDKServer) Run(cmd *cobra.Command, _ []string) error {
	opts, err := c.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	return sdkserver.Start(cmd.Context(), sdkserver.Options{
		Options:       opts,
		ListenAddress: c.ListenAddress,
		Debug:         c.Debug,
	})
}
