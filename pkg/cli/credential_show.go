package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gptscript-ai/gptscript/pkg/gptscript"
	"github.com/spf13/cobra"
)

type Show struct {
	root *GPTScript
}

func (c *Show) Customize(cmd *cobra.Command) {
	cmd.Use = "show <credential name>"
	cmd.Aliases = []string{"reveal"}
	cmd.SilenceUsage = true
	cmd.Short = "Show the secret value of a stored credential"
	cmd.Args = cobra.ExactArgs(1)
}

func (c *Show) Run(cmd *cobra.Command, args []string) error {
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

	cred, exists, err := store.Get(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to get credential: %w", err)
	}

	if !exists {
		return fmt.Errorf("credential %q not found", args[0])
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
	defer w.Flush()

	_, _ = w.Write([]byte("ENV\tVALUE\n"))
	for env, val := range cred.Env {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", env, val)
	}

	return nil
}
