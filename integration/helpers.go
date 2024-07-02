package integration

import (
	"os/exec"
	"runtime"
)

func GPTScriptExec(args ...string) (string, error) {
	cmd := exec.Command("../bin/gptscript", args...)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("..\\bin\\gptscript.exe", args...)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}
