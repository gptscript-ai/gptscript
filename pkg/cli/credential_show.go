package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
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

func (c *Show) Run(_ *cobra.Command, args []string) error {
	opts, err := c.root.NewGPTScriptOpts()
	if err != nil {
		return err
	}
	opts.Cache = cache.Complete(opts.Cache)

	cfg, err := config.ReadCLIConfig(c.root.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read CLI config: %w", err)
	}

	store, err := credentials.NewStore(cfg, c.root.CredentialContext, opts.Cache.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to get credentials store: %w", err)
	}

	cred, exists, err := store.Get(args[0])
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
