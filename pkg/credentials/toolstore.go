package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker-credential-helpers/client"
	credentials2 "github.com/docker/docker-credential-helpers/credentials"
)

type toolCredentialStore struct {
	file     credentials.Store
	program  client.ProgramFunc
	contexts []string
}

func (h *toolCredentialStore) Erase(serverAddress string) error {
	var errs []error
	if err := client.Erase(h.program, serverAddress); err != nil {
		errs = append(errs, err)
	}
	if err := h.file.Erase(serverAddress); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (h *toolCredentialStore) Get(serverAddress string) (types.AuthConfig, error) {
	creds, err := client.Get(h.program, serverAddress)
	if IsCredentialsNotFoundError(err) {
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

// GetAll will list all credentials in the credential store.
// It MAY (but is not required to) filter the credentials based on the contexts provided.
// This is only supported by some credential stores, while others will ignore it and return all credentials.
// The caller of this function is still required to filter the output to only include the contexts requested.
func (h *toolCredentialStore) GetAll() (map[string]types.AuthConfig, error) {
	var (
		serverAddresses map[string]string
		err             error
	)
	if len(h.contexts) == 0 {
		serverAddresses, err = client.List(h.program)
	} else {
		serverAddresses, err = listWithContexts(h.program, h.contexts)
	}

	if err != nil {
		return nil, err
	}

	result := make(map[string]types.AuthConfig, len(serverAddresses))
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

		result[toolNameWithCtx(toolName, ctx)] = types.AuthConfig{
			Username:      val,
			ServerAddress: serverAddress,
		}
	}

	return result, nil
}

func (h *toolCredentialStore) Store(authConfig types.AuthConfig) error {
	return client.Store(h.program, &credentials2.Credentials{
		ServerURL: authConfig.ServerAddress,
		Username:  authConfig.Username,
		Secret:    authConfig.Password,
	})
}

// listWithContexts is almost an exact copy of the List function in Docker's libraries,
// the only difference being that we pass the context through as input to the program.
// This will allow some credential stores, like Postgres, to do an optimized list.
func listWithContexts(program client.ProgramFunc, contexts []string) (map[string]string, error) {
	cmd := program(credentials2.ActionList)

	contextsJSON, err := json.Marshal(contexts)
	if err != nil {
		return nil, err
	}

	cmd.Input(bytes.NewReader(contextsJSON))
	out, err := cmd.Output()
	if err != nil {
		t := strings.TrimSpace(string(out))

		if isValidErr := isValidCredsMessage(t); isValidErr != nil {
			err = isValidErr
		}

		return nil, fmt.Errorf("error listing credentials - err: %v, out: `%s`", err, t)
	}

	var resp map[string]string
	if err = json.NewDecoder(bytes.NewReader(out)).Decode(&resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func isValidCredsMessage(msg string) error {
	if credentials2.IsCredentialsMissingServerURLMessage(msg) {
		return credentials2.NewErrCredentialsMissingServerURL()
	}
	if credentials2.IsCredentialsMissingUsernameMessage(msg) {
		return credentials2.NewErrCredentialsMissingUsername()
	}
	return nil
}
