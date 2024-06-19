package golang

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

	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/debugcmd"
	runtimeEnv "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/download"
)

//go:embed digests.txt
var releasesData []byte

const downloadURL = "https://go.dev/dl/"

type Runtime struct {
	// version something like "1.22.1"
	Version string
}

func (r *Runtime) ID() string {
	return "go" + r.Version
}

func (r *Runtime) Supports(cmd []string) bool {
	return len(cmd) > 0 && cmd[0] == "${GPTSCRIPT_TOOL_DIR}/bin/gptscript-go-tool"
}

func (r *Runtime) Setup(ctx context.Context, dataRoot, toolSource string, env []string) ([]string, error) {
	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return nil, err
	}

	newEnv := runtimeEnv.AppendPath(env, binPath)
	if err := r.runBuild(ctx, toolSource, binPath, append(env, newEnv...)); err != nil {
		return nil, err
	}

	return newEnv, nil
}

func (r *Runtime) BuildCredentialHelper(ctx context.Context, helperName string, credHelperDirs credentials.CredentialHelperDirs, dataRoot, revision string, env []string) error {
	if helperName == "file" {
		return nil
	}

	var suffix string
	if helperName == "wincred" {
		suffix = ".exe"
	}

	binPath, err := r.getRuntime(ctx, dataRoot)
	if err != nil {
		return err
	}
	newEnv := runtimeEnv.AppendPath(env, binPath)

	log.InfofCtx(ctx, "Building credential helper %s", helperName)
	cmd := debugcmd.New(ctx, filepath.Join(binPath, "go"),
		"build", "-buildvcs=false", "-o",
		filepath.Join(credHelperDirs.BinDir, "gptscript-credential-"+helperName+suffix),
		fmt.Sprintf("./%s/cmd/", helperName))
	cmd.Env = stripGo(append(env, newEnv...))
	cmd.Dir = filepath.Join(credHelperDirs.RepoDir, revision)
	return cmd.Run()
}

func (r *Runtime) getReleaseAndDigest() (string, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(releasesData))
	key := r.ID() + "." + runtime.GOOS + "-" + runtime.GOARCH
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "  ")
		file, digest := strings.TrimSpace(line[1]), strings.TrimSpace(line[0])
		if strings.HasPrefix(file, key) {
			return downloadURL + file, digest, nil
		}
	}

	return "", "", fmt.Errorf("failed to find %s release for os=%s arch=%s", r.ID(), runtime.GOOS, runtime.GOARCH)
}

func stripGo(env []string) (result []string) {
	for _, env := range env {
		if strings.HasPrefix(env, "GO") {
			continue
		}
		result = append(result, env)
	}
	return
}

func (r *Runtime) runBuild(ctx context.Context, toolSource, binDir string, env []string) error {
	log.InfofCtx(ctx, "Running go build in %s", toolSource)
	cmd := debugcmd.New(ctx, filepath.Join(binDir, "go"), "build", "-buildvcs=false", "-o", artifactName())
	cmd.Env = stripGo(env)
	cmd.Dir = toolSource
	return cmd.Run()
}

func artifactName() string {
	if runtime.GOOS == "windows" {
		return filepath.Join("bin", "gptscript-go-tool.exe")
	}
	return filepath.Join("bin", "gptscript-go-tool")
}

func (r *Runtime) binDir(rel string) string {
	return filepath.Join(rel, "go", "bin")
}

func (r *Runtime) getRuntime(ctx context.Context, cwd string) (string, error) {
	url, sha, err := r.getReleaseAndDigest()
	if err != nil {
		return "", err
	}

	target := filepath.Join(cwd, "golang", hash.ID(url, sha))
	if _, err := os.Stat(target); err == nil {
		return r.binDir(target), nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	log.InfofCtx(ctx, "Downloading Go %s", r.Version)
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

	return r.binDir(target), nil
}
