package node

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/debugcmd"
	runtimeEnv "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/download"
)

//go:embed SHASUMS256.txt.asc
var releasesData []byte

const downloadURL = "https://nodejs.org/dist/%s/"

type Runtime struct {
	// version something like "3.12"
	Version string
	// If true this is the version that will be used for python or python3
	Default bool
}

func (r *Runtime) ID() string {
	return "node" + r.Version
}

func (r *Runtime) Supports(cmd []string) bool {
	for _, testCmd := range []string{"node", "npx", "npm"} {
		if r.supports(testCmd, cmd) {
			return true
		}
	}
	return false
}

func (r *Runtime) supports(testCmd string, cmd []string) bool {
	if runtimeEnv.Matches(cmd, testCmd+r.Version) {
		return true
	}
	if !r.Default {
		return false
	}
	return runtimeEnv.Matches(cmd, testCmd)
}

func (r *Runtime) Setup(ctx context.Context, dataRoot, toolSource string, env []string) ([]string, error) {
	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return nil, err
	}

	newEnv := runtimeEnv.AppendPath(env, binPath)
	if err := r.runNPM(ctx, toolSource, binPath, append(env, newEnv...)); err != nil {
		return nil, err
	}

	return newEnv, nil
}

func osName() string {
	if runtime.GOOS == "windows" {
		return "win"
	}
	return runtime.GOOS
}

func arch() string {
	if runtime.GOARCH == "amd64" {
		return "x64"
	}
	return runtime.GOARCH
}

func (r *Runtime) getReleaseAndDigest() (string, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(releasesData))
	key := "-" + osName() + "-" + arch()
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "node-v"+r.Version) && strings.Contains(line, key) {
			parts := strings.Split(line, "  ")
			digest := strings.TrimSpace(parts[0])
			file := strings.TrimSpace(parts[1])
			version := strings.Split(file, "-")[1]

			return fmt.Sprintf(downloadURL, version) + file, digest, nil
		}
	}

	return "", "", fmt.Errorf("failed to find %s release for os=%s arch=%s", r.ID(), osName(), arch())
}

func (r *Runtime) runNPM(ctx context.Context, toolSource, binDir string, env []string) error {
	log.InfofCtx(ctx, "Running npm in %s", toolSource)
	cmd := debugcmd.New(ctx, filepath.Join(binDir, "npm"), "install")
	cmd.Env = env
	cmd.Dir = toolSource
	return cmd.Run()
}

func (r *Runtime) binDir(rel string) (string, error) {
	entries, err := os.ReadDir(rel)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(rel, entry.Name(), "node.exe")); err == nil {
				return filepath.Join(rel, entry.Name()), nil
			} else if !errors.Is(err, fs.ErrNotExist) {
				return "", err
			}
			return filepath.Join(rel, entry.Name(), "bin"), nil
		}
	}

	return "", fmt.Errorf("failed to find sub dir for node in %s", rel)
}

func (r *Runtime) getRuntime(ctx context.Context, cwd string) (string, error) {
	url, sha, err := r.getReleaseAndDigest()
	if err != nil {
		return "", err
	}

	target := filepath.Join(cwd, "node", hash.ID(url, sha))
	if _, err := os.Stat(target); err == nil {
		return r.binDir(target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	log.InfofCtx(ctx, "Downloading Node %s.x", r.Version)
	tmp := target + ".download"
	defer os.RemoveAll(tmp)

	if err := os.MkdirAll(tmp, 0755); err != nil {
		return "", err
	}

	if err := download.Extract(ctx, url, sha, tmp); err != nil {
		return "", err
	}

	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}

	return r.binDir(target)
}
