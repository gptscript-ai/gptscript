package system

import "os"

const BinEnvVar = "GPTSCRIPT_BIN"

func Bin() string {
	bin := os.Getenv(BinEnvVar)
	if bin != "" {
		return bin
	}
	return currentBin()
}
