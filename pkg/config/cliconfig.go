package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/docker/cli/cli/config/types"
)

var (
	darwinHelpers  = []string{"osxkeychain", "file"}
	windowsHelpers = []string{"wincred", "file"}
	linuxHelpers   = []string{"secretservice", "pass", "file"}
)

const GPTScriptHelperPrefix = "gptscript-credential-"

type AuthConfig types.AuthConfig

func (a AuthConfig) MarshalJSON() ([]byte, error) {
	cp := a
	if cp.Username != "" || cp.Password != "" {
		cp.Auth = base64.StdEncoding.EncodeToString([]byte(cp.Username + ":" + cp.Password))
		cp.Username = ""
		cp.Password = ""
	}
	cp.ServerAddress = ""
	return json.Marshal((types.AuthConfig)(cp))
}

func (a *AuthConfig) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, (*types.AuthConfig)(a)); err != nil {
		return err
	}
	if a.Auth != "" {
		data, err := base64.StdEncoding.DecodeString(a.Auth)
		if err != nil {
			return err
		}
		a.Username, a.Password, _ = strings.Cut(string(data), ":")
		a.Auth = ""
	}
	return nil
}

type CLIConfig struct {
	Auths               map[string]AuthConfig `json:"auths,omitempty"`
	CredentialsStore    string                `json:"credsStore,omitempty"`
	GPTScriptConfigFile string                `json:"gptscriptConfig,omitempty"`

	auths     map[string]types.AuthConfig
	authsLock *sync.Mutex
}

func (c *CLIConfig) Sanitize() *CLIConfig {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Auths = map[string]AuthConfig{}
	for k := range c.Auths {
		cp.Auths[k] = AuthConfig{
			Auth: "<redacted>",
		}
	}
	return &cp
}

func (c *CLIConfig) Save() error {
	if c.authsLock != nil {
		c.authsLock.Lock()
		defer c.authsLock.Unlock()
	}

	if c.auths != nil {
		c.Auths = map[string]AuthConfig{}
		for k, v := range c.auths {
			c.Auths[k] = (AuthConfig)(v)
		}
		c.auths = nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.GPTScriptConfigFile, data, 0655)
}

func (c *CLIConfig) GetAuthConfigs() map[string]types.AuthConfig {
	if c.authsLock != nil {
		c.authsLock.Lock()
		defer c.authsLock.Unlock()
	}

	if c.auths == nil {
		c.auths = map[string]types.AuthConfig{}
		for k, v := range c.Auths {
			authConfig := (types.AuthConfig)(v)
			c.auths[k] = authConfig
		}
	}
	return c.auths
}

func (c *CLIConfig) GetFilename() string {
	return c.GPTScriptConfigFile
}

func ReadCLIConfig(gptscriptConfigFile string) (*CLIConfig, error) {
	if gptscriptConfigFile == "" {
		// If gptscriptConfigFile isn't provided, check the environment variable
		if gptscriptConfigFile = os.Getenv("GPTSCRIPT_CONFIG_FILE"); gptscriptConfigFile == "" {
			// If an environment variable isn't provided, check the default location
			var err error
			if gptscriptConfigFile, err = xdg.ConfigFile("gptscript/config.json"); err != nil {
				return nil, fmt.Errorf("failed to read user config from standard location: %w", err)
			}
		}
	}

	data, err := readFile(gptscriptConfigFile)
	if err != nil {
		return nil, err
	}
	result := &CLIConfig{
		authsLock:           &sync.Mutex{},
		GPTScriptConfigFile: gptscriptConfigFile,
	}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, err
	}

	if result.CredentialsStore == "" {
		if err := result.setDefaultCredentialsStore(); err != nil {
			return nil, err
		}
	}

	if !isValidCredentialHelper(result.CredentialsStore) {
		errMsg := fmt.Sprintf("invalid credential store '%s'", result.CredentialsStore)
		switch runtime.GOOS {
		case "darwin":
			errMsg += " (use 'osxkeychain' or 'file')"
		case "windows":
			errMsg += " (use 'wincred' or 'file')"
		case "linux":
			errMsg += " (use 'secretservice', 'pass', or 'file')"
		default:
			errMsg += " (use 'file')"
		}
		errMsg += fmt.Sprintf("\nPlease edit your config file at %s to fix this.", result.GPTScriptConfigFile)

		return nil, errors.New(errMsg)
	}

	return result, nil
}

func (c *CLIConfig) setDefaultCredentialsStore() error {
	switch runtime.GOOS {
	case "darwin":
		c.CredentialsStore = "osxkeychain"
	case "windows":
		c.CredentialsStore = "wincred"
	default:
		c.CredentialsStore = "file"
	}
	return c.Save()
}

func isValidCredentialHelper(helper string) bool {
	switch runtime.GOOS {
	case "darwin":
		return slices.Contains(darwinHelpers, helper)
	case "windows":
		return slices.Contains(windowsHelpers, helper)
	case "linux":
		return slices.Contains(linuxHelpers, helper)
	default:
		return helper == "file"
	}
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []byte("{}"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read user config %s: %w", path, err)
	}

	return data, nil
}
