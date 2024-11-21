package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"github.com/docker/cli/cli/config/types"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
)

const (
	WincredCredHelper       = "wincred"
	OsxkeychainCredHelper   = "osxkeychain"
	SecretserviceCredHelper = "secretservice"
	PassCredHelper          = "pass"
	FileCredHelper          = "file"
)

var (
	// Helpers is a list of all supported credential helpers from github.com/gptscript-ai/gptscript-credential-helpers
	Helpers = []string{WincredCredHelper, OsxkeychainCredHelper, SecretserviceCredHelper, PassCredHelper}
	log     = mvl.Package()
)

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
	Auths            map[string]AuthConfig `json:"auths,omitempty"`
	CredentialsStore string                `json:"credsStore,omitempty"`

	raw       []byte
	auths     map[string]types.AuthConfig
	authsLock *sync.Mutex
	location  string
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
		c.Auths = make(map[string]AuthConfig, len(c.auths))
		for k, v := range c.auths {
			c.Auths[k] = AuthConfig(v)
		}
		c.auths = nil
	}

	// This is to not overwrite additional fields that might be the config file
	out := map[string]any{}
	if len(c.raw) > 0 {
		err := json.Unmarshal(c.raw, &out)
		if err != nil {
			return err
		}
	}
	out["auths"] = c.Auths
	out["credsStore"] = c.CredentialsStore

	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(c.location, data, 0655)
}

func (c *CLIConfig) GetAuthConfigs() map[string]types.AuthConfig {
	if c.authsLock != nil {
		c.authsLock.Lock()
		defer c.authsLock.Unlock()
	}

	if err := c.readFileIntoConfig(c.location); err != nil {
		// This is implementing an interface, so we can't return this error.
		log.Warnf("Failed to read config file: %v", err)
	}

	if c.auths == nil {
		c.auths = make(map[string]types.AuthConfig, len(c.Auths))
	}

	// Assume that whatever was pulled from the file is more recent.
	// The docker creds framework will save the file after creating or updating a credential.
	for k, v := range c.Auths {
		c.auths[k] = types.AuthConfig(v)
	}

	return c.auths
}

func (c *CLIConfig) GetFilename() string {
	return c.location
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

	result := &CLIConfig{
		authsLock: &sync.Mutex{},
		location:  gptscriptConfigFile,
	}

	if err := result.readFileIntoConfig(gptscriptConfigFile); err != nil {
		return nil, err
	}

	if store := os.Getenv("GPTSCRIPT_CREDENTIAL_STORE"); store != "" {
		result.CredentialsStore = store
	}

	if result.CredentialsStore == "" {
		if err := result.setDefaultCredentialsStore(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (c *CLIConfig) setDefaultCredentialsStore() error {
	switch runtime.GOOS {
	case "darwin":
		c.CredentialsStore = OsxkeychainCredHelper
	case "windows":
		c.CredentialsStore = WincredCredHelper
	default:
		c.CredentialsStore = FileCredHelper
	}
	return c.Save()
}

func (c *CLIConfig) readFileIntoConfig(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read user config %s: %w", path, err)
	}

	c.raw = data
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to unmarshal %s: %v", path, err)
	}

	return nil
}
