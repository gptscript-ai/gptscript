package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func execEquals(bin, check string) bool {
	return bin == check ||
		bin == check+".exe"
}

func VarOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}

func ToEnvLike(v string) string {
	v = strings.ReplaceAll(v, ".", "_")
	v = strings.ReplaceAll(v, "-", "_")
	return strings.ToUpper(v)
}

func Matches(cmd []string, bin string) bool {
	switch len(cmd) {
	case 0:
		return false
	case 1:
		return execEquals(cmd[0], bin)
	}
	if cmd[0] == bin {
		return true
	}
	if cmd[0] == "/usr/bin/env" || cmd[0] == "/bin/env" {
		return execEquals(cmd[1], bin)
	}
	return false
}

func AppendPath(env []string, binPath string) []string {
	var newEnv []string
	for _, path := range env {
		v, ok := strings.CutPrefix(path, "PATH=")
		if ok {
			newEnv = append(newEnv, fmt.Sprintf("PATH=%s%s%s",
				binPath, string(os.PathListSeparator), v))
		} else if v, ok := strings.CutPrefix(path, "Path="); ok {
			newEnv = append(newEnv, fmt.Sprintf("Path=%s%s%s",
				binPath, string(os.PathListSeparator), v))
		}
	}
	return newEnv
}

// Lookup will try to find bin in the PATH in env. It will refer to PATHEXT for Windows support.
// If bin can not be resolved to anything the original bin string is returned.
func Lookup(env []string, bin string) string {
	if strings.Contains(bin, string(filepath.Separator)) {
		return bin
	}

	for _, env := range env {
		for _, prefix := range []string{"PATH=", "Path="} {
			suffix, ok := strings.CutPrefix(env, prefix)
			if !ok {
				continue
			}
			log.Debugf("Looking for %s in %s", bin, suffix)
			for _, path := range strings.Split(suffix, string(os.PathListSeparator)) {
				testPath := filepath.Join(path, bin)

				if stat, err := os.Stat(testPath); err == nil && !stat.IsDir() {
					log.Debugf("Found %s for %s in %s", testPath, bin, suffix)
					return testPath
				}

				for _, ext := range strings.Split(os.Getenv("PATHEXT"), string(os.PathListSeparator)) {
					if ext == "" {
						continue
					}

					if stat, err := os.Stat(testPath + ext); err == nil && !stat.IsDir() {
						log.Debugf("Found %s for %s in %s", testPath+ext, bin, suffix)
						return testPath + ext
					}
				}
			}
		}
	}

	return bin
}
