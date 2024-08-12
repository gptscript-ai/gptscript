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
	"github.com/gptscript-ai/gptscript/pkg/types"
)

//go:embed SHASUMS256.txt.asc
var releasesData []byte

const (
	downloadURL = "https://nodejs.org/dist/%s/"
	packageJSON = "package.json"
)

type Runtime struct {
	// version something like "3.12"
	Version string
	// If true this is the version that will be used for python or python3
	Default bool
}

func (r *Runtime) ID() string {
	return "node" + r.Version
}

func (r *Runtime) Binary(_ context.Context, _ types.Tool, _, _ string, _ []string) (bool, []string, error) {
	return false, nil, nil
}

func (r *Runtime) Supports(_ types.Tool, cmd []string) bool {
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

func (r *Runtime) GetHash(tool types.Tool) (string, error) {
	if !tool.Source.IsGit() && tool.WorkingDir != "" {
		if s, err := os.Stat(filepath.Join(tool.WorkingDir, packageJSON)); err == nil {
			return hash.Digest(tool.WorkingDir + s.ModTime().String())[:7], nil
		}
	}
	return "", nil
}

func (r *Runtime) Setup(ctx context.Context, tool types.Tool, dataRoot, toolSource string, env []string) ([]string, error) {
	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return nil, err
	}

	newEnv := runtimeEnv.AppendPath(env, binPath)
	if err := r.runNPM(ctx, tool, toolSource, binPath, append(env, newEnv...)); err != nil {
		return nil, err
	}

	if _, ok := tool.MetaData[packageJSON]; ok {
		newEnv = append(newEnv, "GPTSCRIPT_TMPDIR="+toolSource)
	} else if !tool.Source.IsGit() && tool.WorkingDir != "" {
		newEnv = append(newEnv, "GPTSCRIPT_TMPDIR="+tool.WorkingDir, "GPTSCRIPT_RUNTIME_DEV=true")
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

func (r *Runtime) runNPM(ctx context.Context, tool types.Tool, toolSource, binDir string, env []string) error {
	log.InfofCtx(ctx, "Running npm in %s", toolSource)
	cmd := debugcmd.New(ctx, filepath.Join(binDir, "npm"), "install")
	cmd.Env = env
	cmd.Dir = toolSource
	if contents, ok := tool.MetaData[packageJSON]; ok {
		if err := os.WriteFile(filepath.Join(toolSource, packageJSON), []byte(contents+"\n"), 0644); err != nil {
			return err
		}
	} else if !tool.Source.IsGit() {
		if tool.WorkingDir == "" {
			return nil
		}
		if _, err := os.Stat(filepath.Join(tool.WorkingDir, packageJSON)); errors.Is(fs.ErrNotExist, err) {
			return nil
		} else if err != nil {
			return err
		}
		cmd.Dir = tool.WorkingDir
	}
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
