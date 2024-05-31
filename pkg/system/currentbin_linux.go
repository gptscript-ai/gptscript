//go:build linux

package system

func currentBin() string {
	// Linux is simple, always use this path
	return "/proc/self/exe"
}
