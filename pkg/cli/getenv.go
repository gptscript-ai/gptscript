package cli

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type Getenv struct {
}

func (e *Getenv) Customize(cmd *cobra.Command) {
	cmd.Use = "getenv [flags] KEY [DEFAULT]"
	cmd.Short = "Looks up an environment variable for use in GPTScript tools"
	cmd.Args = cobra.RangeArgs(1, 2)
}

func (e *Getenv) Run(_ *cobra.Command, args []string) error {
	var (
		key = args[0]
		def string
	)
	if len(args) > 1 {
		def = args[1]
	}
	value := getEnv(key, def)
	fmt.Print(value)
	return nil
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	if strings.HasPrefix(v, `{"_gz":"`) && strings.HasSuffix(v, `"}`) {
		data, err := base64.StdEncoding.DecodeString(v[8 : len(v)-2])
		if err != nil {
			return v
		}
		gz, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			return v
		}
		strBytes, err := io.ReadAll(gz)
		if err != nil {
			return v
		}
		return string(strBytes)
	}

	return v
}
