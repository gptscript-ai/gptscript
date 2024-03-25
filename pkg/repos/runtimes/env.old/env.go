package env

import (
	"fmt"
	"os"
	"strings"
)

func execEquals(bin, check string) bool {
	return bin == check ||
		bin == check+".exe"
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
		for _, prefix := range []string{"PATH=", "Path="} {
			v, ok := strings.CutPrefix(path, prefix)
			if ok {
				newEnv = append(newEnv, fmt.Sprintf(prefix+"%s%s%s",
					binPath, string(os.PathListSeparator), v))
			}
		}
	}
	return newEnv
}
