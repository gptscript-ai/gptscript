package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	cmd2 "github.com/gptscript-ai/cmd"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/gptscript"
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
	opts, err := c.root.NewGPTScriptOpts()
	if err != nil {
		return err
	}
	gptScript, err := gptscript.New(cmd.Context(), opts)
	if err != nil {
		return err
	}
	defer gptScript.Close(true)

	credCtxs := gptScript.DefaultCredentialContexts
	if c.AllContexts {
		credCtxs = []string{credentials.AllCredentialContexts}
	}

	store, err := gptScript.CredentialStoreFactory.NewStore(credCtxs)
	if err != nil {
		return err
	}

	creds, err := store.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
	defer w.Flush()

	// Sort credentials and print column names, depending on the options.
	if c.AllContexts || len(c.root.CredentialContext) > 1 {
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
		if c.AllContexts || len(c.root.CredentialContext) > 1 {
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
