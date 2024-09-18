package credentials

import (
	"path/filepath"
)

type CredentialHelperDirs struct {
	RevisionFile, LastCheckedFile, BinDir string
}

func GetCredentialHelperDirs(cacheDir string) CredentialHelperDirs {
	return CredentialHelperDirs{
		RevisionFile:    filepath.Join(cacheDir, "repos", "gptscript-credential-helpers", "revision"),
		LastCheckedFile: filepath.Join(cacheDir, "repos", "gptscript-credential-helpers", "last-checked"),
		BinDir:          filepath.Join(cacheDir, "repos", "gptscript-credential-helpers", "bin"),
	}
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
