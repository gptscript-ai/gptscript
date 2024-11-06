package credentials

import (
	"strings"
)

func IsCredentialsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "credentials not found in native keychain")
}
