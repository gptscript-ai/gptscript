package cli

import (
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/spf13/cobra"
)

type Delete struct {
	root *GPTScript
}

func (c *Delete) Customize(cmd *cobra.Command) {
	cmd.Use = "delete <credential name>"
	cmd.Aliases = []string{"rm", "del"}
	cmd.SilenceUsage = true
	cmd.Short = "Delete a stored credential"
	cmd.Args = cobra.ExactArgs(1)
}

func (c *Delete) Run(cmd *cobra.Command, args []string) error {
	opts, err := c.root.NewGPTScriptOpts()
	if err != nil {
		return err
	}

	gptScript, err := gptscript.New(cmd.Context(), opts)
	if err != nil {
		return err
	}
	defer gptScript.Close(true)

	store, err := gptScript.CredentialStoreFactory.NewStore(gptScript.DefaultCredentialContexts)
	if err != nil {
		return err
	}

	if err = store.Remove(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("failed to remove credential: %w", err)
	}
	return nil
}
