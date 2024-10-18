package busybox

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	runtimeEnv "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/download"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

//go:embed SHASUMS256.txt
var releasesData []byte

const downloadURL = "https://github.com/gptscript-ai/busybox-w32/releases/download/%s"

type Runtime struct {
	runtimeSetupLock sync.Mutex
}

func (r *Runtime) ID() string {
	return "busybox"
}

func (r *Runtime) GetHash(_ types.Tool) (string, error) {
	return "", nil
}

func (r *Runtime) Supports(_ types.Tool, cmd []string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	for _, bin := range []string{"bash", "sh", "/bin/sh", "/bin/bash"} {
		if runtimeEnv.Matches(cmd, bin) {
			return true
		}
	}
	return false
}

func (r *Runtime) Binary(_ context.Context, _ types.Tool, _, _ string, _ []string) (bool, []string, error) {
	return false, nil, nil
}

func (r *Runtime) Setup(ctx context.Context, _ types.Tool, dataRoot, _ string, env []string) ([]string, error) {
	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return nil, err
	}

	newEnv := runtimeEnv.AppendPath(env, binPath)
	return newEnv, nil
}

func (r *Runtime) getReleaseAndDigest() (string, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(releasesData))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		return fmt.Sprintf(downloadURL, fields[1]), fields[0], nil
	}

	return "", "", fmt.Errorf("failed to find %s release", r.ID())
}

func (r *Runtime) getRuntime(ctx context.Context, cwd string) (string, error) {
	r.runtimeSetupLock.Lock()
	defer r.runtimeSetupLock.Unlock()

	url, sha, err := r.getReleaseAndDigest()
	if err != nil {
		return "", err
	}

	target := filepath.Join(cwd, "busybox", hash.ID(url, sha))
	if _, err := os.Stat(target); err == nil {
		return target, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	log.Infof("Downloading Busybox")
	tmp := target + ".download"
	defer os.RemoveAll(tmp)

	if err := os.MkdirAll(tmp, 0755); err != nil {
		return "", err
	}

	if err := download.Extract(ctx, url, sha, tmp); err != nil {
		return "", err
	}

	bbExe := filepath.Join(tmp, path.Base(url))

	cmd := exec.Command(bbExe, "--install", ".")
	cmd.Dir = filepath.Dir(bbExe)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}

	return target, nil
}
