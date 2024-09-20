package credentials

import (
	"fmt"
	"path/filepath"

	runtimeEnv "github.com/gptscript-ai/gptscript/pkg/env"
)

type CredentialHelperDirs struct {
	RevisionFile, LastCheckedFile, BinDir string
}

func RepoNameForCredentialStore(store string) string {
	switch store {
	case "sqlite":
		return "gptscript-credential-sqlite"
	default:
		return "gptscript-credential-helpers"
	}
}

func GitURLForRepoName(repoName string) (string, error) {
	switch repoName {
	case "gptscript-credential-sqlite":
		return runtimeEnv.VarOrDefault("GPTSCRIPT_CRED_SQLITE_ROOT", "https://github.com/gptscript-ai/gptscript-credential-sqlite.git"), nil
	case "gptscript-credential-helpers":
		return runtimeEnv.VarOrDefault("GPTSCRIPT_CRED_HELPERS_ROOT", "https://github.com/gptscript-ai/gptscript-credential-helpers.git"), nil
	default:
		return "", fmt.Errorf("unknown repo name: %s", repoName)
	}
}

func GetCredentialHelperDirs(cacheDir, store string) CredentialHelperDirs {
	repoName := RepoNameForCredentialStore(store)
	return CredentialHelperDirs{
		RevisionFile:    filepath.Join(cacheDir, "repos", repoName, "revision"),
		LastCheckedFile: filepath.Join(cacheDir, "repos", repoName, "last-checked"),
		BinDir:          filepath.Join(cacheDir, "repos", repoName, "bin"),
	}
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
