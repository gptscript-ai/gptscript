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
