package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	cmd2 "github.com/gptscript-ai/cmd"
	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/spf13/cobra"
)

const (
	expiresNever   = "never"
	expiresExpired = "expired"
)

type Credential struct {
	root        *GPTScript
	AllContexts bool `usage:"List credentials for all contexts" local:"true"`
	ShowEnvVars bool `usage:"Show names of environment variables in each credential" local:"true"`
}

func (c *Credential) Customize(cmd *cobra.Command) {
	cmd.Use = "credential"
	cmd.Aliases = []string{"cred", "creds", "credentials"}
	cmd.Short = "List stored credentials"
	cmd.Args = cobra.NoArgs
	cmd.AddCommand(cmd2.Command(&Delete{root: c.root}))
	cmd.AddCommand(cmd2.Command(&Show{root: c.root}))
}

func (c *Credential) Run(cmd *cobra.Command, _ []string) error {
	cfg, err := config.ReadCLIConfig(c.root.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read CLI config: %w", err)
	}

	ctx := c.root.CredentialContext
	if c.AllContexts {
		ctx = "*"
	}

	opts, err := c.root.NewGPTScriptOpts()
	if err != nil {
		return err
	}
	opts.Cache = cache.Complete(opts.Cache)
	opts.Runner = runner.Complete(opts.Runner)
	if opts.Runner.RuntimeManager == nil {
		opts.Runner.RuntimeManager = runtimes.Default(opts.Cache.CacheDir)
	}

	if err = opts.Runner.RuntimeManager.SetUpCredentialHelpers(cmd.Context(), cfg, opts.Env); err != nil {
		return err
	}

	// Initialize the credential store and get all the credentials.
	store, err := credentials.NewStore(cfg, opts.Runner.RuntimeManager, ctx, opts.Cache.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to get credentials store: %w", err)
	}

	creds, err := store.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
	defer w.Flush()

	// Sort credentials and print column names, depending on the options.
	if c.AllContexts {
		// Sort credentials by context
		sort.Slice(creds, func(i, j int) bool {
			if creds[i].Context == creds[j].Context {
				return creds[i].ToolName < creds[j].ToolName
			}
			return creds[i].Context < creds[j].Context
		})

		if c.ShowEnvVars {
			_, _ = w.Write([]byte("CONTEXT\tCREDENTIAL\tEXPIRES IN\tENV\n"))
		} else {
			_, _ = w.Write([]byte("CONTEXT\tCREDENTIAL\tEXPIRES IN\n"))
		}
	} else {
		// Sort credentials by tool name
		sort.Slice(creds, func(i, j int) bool {
			return creds[i].ToolName < creds[j].ToolName
		})

		if c.ShowEnvVars {
			_, _ = w.Write([]byte("CREDENTIAL\tEXPIRES IN\tENV\n"))
		} else {
			_, _ = w.Write([]byte("CREDENTIAL\tEXPIRES IN\n"))
		}
	}

	for _, cred := range creds {
		expires := expiresNever
		if cred.ExpiresAt != nil {
			expires = expiresExpired
			if !cred.IsExpired() {
				expires = time.Until(*cred.ExpiresAt).Truncate(time.Second).String()
			}
		}

		var fields []any
		if c.AllContexts {
			fields = []any{cred.Context, cred.ToolName, expires}
		} else {
			fields = []any{cred.ToolName, expires}
		}

		if c.ShowEnvVars {
			envVars := make([]string, 0, len(cred.Env))
			for envVar := range cred.Env {
				envVars = append(envVars, envVar)
			}
			sort.Strings(envVars)
			fields = append(fields, strings.Join(envVars, ", "))
		}

		printFields(w, fields)
	}

	return nil
}

func printFields(w *tabwriter.Writer, fields []any) {
	if len(fields) == 0 {
		return
	}

	fmtStr := strings.Repeat("%s\t", len(fields)-1) + "%s\n"
	_, _ = fmt.Fprintf(w, fmtStr, fields...)
}
