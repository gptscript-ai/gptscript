package env

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
