package credentials

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker-credential-helpers/client"
	credentials2 "github.com/docker/docker-credential-helpers/credentials"
	"github.com/gptscript-ai/gptscript/pkg/config"
)

func NewHelper(c *config.CLIConfig, helper string) (credentials.Store, error) {
	return &HelperStore{
		file:    credentials.NewFileStore(c),
		program: client.NewShellProgramFunc(helper),
	}, nil
}

type HelperStore struct {
	file    credentials.Store
	program client.ProgramFunc
}

func (h *HelperStore) Erase(serverAddress string) error {
	var errs []error
	if err := client.Erase(h.program, serverAddress); err != nil {
		errs = append(errs, err)
	}
	if err := h.file.Erase(serverAddress); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (h *HelperStore) Get(serverAddress string) (types.AuthConfig, error) {
	creds, err := client.Get(h.program, serverAddress)
	if credentials2.IsErrCredentialsNotFound(err) {
		return h.file.Get(serverAddress)
	} else if err != nil {
		return types.AuthConfig{}, err
	}
	return types.AuthConfig{
		Username:      creds.Username,
		Password:      creds.Secret,
		ServerAddress: serverAddress,
	}, nil
}

func (h *HelperStore) GetAll() (map[string]types.AuthConfig, error) {
	result := map[string]types.AuthConfig{}

	serverAddresses, err := client.List(h.program)
	if err != nil {
		return nil, err
	}

	newCredAddresses := make(map[string]string, len(serverAddresses))
	for serverAddress, val := range serverAddresses {
		// If the serverAddress contains a port, we need to put it back in the right spot.
		// For some reason, even when a credential is stored properly as http://hostname:8080///credctx,
		// the list function will return http://hostname///credctx:8080. This is something wrong
		// with macOS's built-in libraries. So we need to fix it here.
		toolName, ctx, err := toolNameAndCtxFromAddress(serverAddress)
		if err != nil {
			return nil, err
		}

		contextPieces := strings.Split(ctx, ":")
		if len(contextPieces) > 1 {
			possiblePortNumber := contextPieces[len(contextPieces)-1]
			if regexp.MustCompile(`^\d+$`).MatchString(possiblePortNumber) {
				// port number confirmed
				toolURL, err := url.Parse(toolName)
				if err != nil {
					return nil, err
				}

				// Save the path so we can put it back after removing it.
				path := toolURL.Path
				toolURL.Path = ""

				toolName = toolURL.String() + ":" + possiblePortNumber + path
				ctx = strings.TrimSuffix(ctx, ":"+possiblePortNumber)
			}
		}

		newCredAddresses[toolNameWithCtx(toolName, ctx)] = val
		delete(serverAddresses, serverAddress)
	}

	for serverAddress := range newCredAddresses {
		ac, err := h.Get(serverAddress)
		if err != nil {
			return nil, err
		}
		result[serverAddress] = ac
	}

	return result, nil
}

func (h *HelperStore) Store(authConfig types.AuthConfig) error {
	return client.Store(h.program, &credentials2.Credentials{
		ServerURL: authConfig.ServerAddress,
		Username:  authConfig.Username,
		Secret:    authConfig.Password,
	})
}
