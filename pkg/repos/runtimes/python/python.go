package python

import (
	"context"
	// For embedded python.json
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gptscript-ai/gptscript/pkg/debugcmd"
	runtimeEnv "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/download"
)

//go:embed python.json
var releasesData []byte

const uvVersion = "uv==0.1.15"

type Release struct {
	OS      string `json:"os,omitempty"`
	Arch    string `json:"arch,omitempty"`
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

type Runtime struct {
	// version something like "3.12"
	Version string
	// If true this is the version that will be used for python or python3
	Default bool
}

func (r *Runtime) ID() string {
	return "python" + r.Version
}

func (r *Runtime) Supports(cmd []string) bool {
	if runtimeEnv.Matches(cmd, r.ID()) {
		return true
	}
	if !r.Default {
		return false
	}
	return runtimeEnv.Matches(cmd, "python") || runtimeEnv.Matches(cmd, "python3")
}

func (r *Runtime) installVenv(ctx context.Context, binDir, venvPath string) error {
	log.Infof("Creating virtualenv in %s", venvPath)
	cmd := debugcmd.New(ctx, filepath.Join(binDir, "uv"), "venv", "-p",
		filepath.Join(binDir, "python3"), venvPath)
	return cmd.Run()
}

func (r *Runtime) Setup(ctx context.Context, dataRoot, toolSource string, env []string) ([]string, error) {
	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return nil, err
	}

	venvPath := filepath.Join(dataRoot, "venv", hash.ID(binPath, toolSource))
	venvBinPath := filepath.Join(venvPath, "bin")

	// Cleanup failed runs
	if err := os.RemoveAll(venvPath); err != nil {
		return nil, err
	}

	if err := r.installVenv(ctx, binPath, venvPath); err != nil {
		return nil, err
	}

	newEnv := runtimeEnv.AppendPath(env, venvBinPath)
	newEnv = append(newEnv, "VIRTUAL_ENV="+venvPath)

	if err := r.runPip(ctx, toolSource, binPath, append(env, newEnv...)); err != nil {
		return nil, err
	}

	return newEnv, nil
}

func readRelease() (result []Release) {
	if err := json.Unmarshal(releasesData, &result); err != nil {
		panic(err)
	}
	return
}

func (r *Runtime) getReleaseAndDigest() (string, string, error) {
	for _, release := range readRelease() {
		if release.OS == runtime.GOOS &&
			release.Arch == runtime.GOARCH &&
			release.Version == r.Version {
			return release.URL, release.Digest, nil
		}
	}
	return "", "", fmt.Errorf("failed to find an python runtime for %s", r.Version)
}

func (r *Runtime) runPip(ctx context.Context, toolSource, binDir string, env []string) error {
	log.Infof("Running pip in %s", toolSource)
	for _, req := range []string{"requirements-gptscript.txt", "requirements.txt"} {
		reqFile := filepath.Join(toolSource, req)
		if s, err := os.Stat(reqFile); err == nil && !s.IsDir() {
			cmd := debugcmd.New(ctx, filepath.Join(binDir, "uv"), "pip", "install", "-r", reqFile)
			cmd.Env = env
			return cmd.Run()
		}
	}

	return nil
}

func (r *Runtime) setupUV(ctx context.Context, tmp string) error {
	cmd := debugcmd.New(ctx, filepath.Join(tmp, "python", "bin", "python3"),
		filepath.Join(tmp, "python", "bin", "pip"),
		"install", uvVersion)
	return cmd.Run()
}

func (r *Runtime) getRuntime(ctx context.Context, cwd string) (string, error) {
	url, sha, err := r.getReleaseAndDigest()
	if err != nil {
		return "", err
	}

	target := filepath.Join(cwd, "python", hash.ID(url, sha, uvVersion))
	binDir := filepath.Join(target, "python", "bin")
	if _, err := os.Stat(target); err == nil {
		return binDir, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	log.Infof("Downloading Python %s.x", r.Version)
	tmp := target + ".download"
	defer os.RemoveAll(tmp)

	if err := os.MkdirAll(tmp, 0755); err != nil {
		return "", err
	}

	if err := download.Extract(ctx, url, sha, tmp); err != nil {
		return "", err
	}

	if err := r.setupUV(ctx, tmp); err != nil {
		return "", err
	}

	return binDir, os.Rename(tmp, target)
}
