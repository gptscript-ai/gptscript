package system

import "os"

func Bin() string {
	bin := os.Getenv("GPTSCRIPT_BIN")
	if bin != "" {
		return bin
	}
	return currentBin()
}
