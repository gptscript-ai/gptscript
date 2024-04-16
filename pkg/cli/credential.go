package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	cmd2 "github.com/acorn-io/cmd"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/spf13/cobra"
)

type Credential struct {
	root        *GPTScript
	AllContexts bool `usage:"List credentials for all contexts" local:"true"`
}

func (c *Credential) Customize(cmd *cobra.Command) {
	cmd.Use = "credential"
	cmd.Aliases = []string{"cred", "creds", "credentials"}
	cmd.Short = "List stored credentials"
	cmd.Args = cobra.NoArgs
	cmd.AddCommand(cmd2.Command(&Delete{root: c.root}))
}

func (c *Credential) Run(_ *cobra.Command, _ []string) error {
	cfg, err := config.ReadCLIConfig(c.root.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read CLI config: %w", err)
	}

	ctx := c.root.CredentialContext
	if c.AllContexts {
		ctx = "*"
	}

	store, err := credentials.NewStore(cfg, ctx)
	if err != nil {
		return fmt.Errorf("failed to get credentials store: %w", err)
	}

	creds, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	if c.AllContexts {
		// Sort credentials by context
		sort.Slice(creds, func(i, j int) bool {
			if creds[i].Context == creds[j].Context {
				return creds[i].ToolName < creds[j].ToolName
			}
			return creds[i].Context < creds[j].Context
		})

		w := tabwriter.NewWriter(os.Stdout, 10, 1, 3, ' ', 0)
		defer w.Flush()
		_, _ = w.Write([]byte("CONTEXT\tTOOL\n"))
		for _, cred := range creds {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", cred.Context, cred.ToolName)
		}
	} else {
		// Sort credentials by tool name
		sort.Slice(creds, func(i, j int) bool {
			return creds[i].ToolName < creds[j].ToolName
		})

		for _, cred := range creds {
			fmt.Println(cred.ToolName)
		}
	}

	return nil
}
