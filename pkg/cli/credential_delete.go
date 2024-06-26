package cli

import (
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes"
	"github.com/gptscript-ai/gptscript/pkg/runner"
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

	cfg, err := config.ReadCLIConfig(c.root.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read CLI config: %w", err)
	}

	opts.Cache = cache.Complete(opts.Cache)
	opts.Runner = runner.Complete(opts.Runner)
	if opts.Runner.RuntimeManager == nil {
		opts.Runner.RuntimeManager = runtimes.Default(opts.Cache.CacheDir)
	}

	if err = opts.Runner.RuntimeManager.SetUpCredentialHelpers(cmd.Context(), cfg, opts.Env); err != nil {
		return err
	}

	store, err := credentials.NewStore(cfg, opts.Runner.RuntimeManager, c.root.CredentialContext, opts.Cache.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to get credentials store: %w", err)
	}

	if err = store.Remove(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("failed to remove credential: %w", err)
	}
	return nil
}
