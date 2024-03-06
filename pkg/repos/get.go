package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gptscript-ai/gptscript/pkg/repos/git"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Runtime interface {
	ID() string
	Supports(cmd []string) bool
	Setup(ctx context.Context, dataRoot, toolSource string, env []string) ([]string, error)
}

type noopRuntime struct {
}

func (n noopRuntime) ID() string {
	return "none"
}

func (n noopRuntime) Supports(_ []string) bool {
	return false
}

func (n noopRuntime) Setup(_ context.Context, _, _ string, _ []string) ([]string, error) {
	return nil, nil
}

type Manager struct {
	storageDir string
	gitDir     string
	runtimeDir string
	runtimes   []Runtime
}

func New(cacheDir string, runtimes ...Runtime) *Manager {
	root := filepath.Join(cacheDir, "repos")
	return &Manager{
		storageDir: root,
		gitDir:     filepath.Join(root, "git"),
		runtimeDir: filepath.Join(root, "runtimes"),
		runtimes:   runtimes,
	}
}

func (m *Manager) setup(ctx context.Context, runtime Runtime, tool types.Tool, env []string) (string, []string, error) {
	target := filepath.Join(m.storageDir, tool.Source.Repo.Revision, runtime.ID())
	doneFile := target + ".done"
	envData, err := os.ReadFile(doneFile)
	if err == nil {
		var savedEnv []string
		if err := json.Unmarshal(envData, &savedEnv); err == nil {
			return target, append(env, savedEnv...), nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", nil, err
	}

	// Cleanup previous failed runs
	_ = os.RemoveAll(doneFile + ".tmp")
	_ = os.RemoveAll(doneFile)
	_ = os.RemoveAll(target)

	if err := git.Checkout(ctx, m.gitDir, tool.Source.Repo.Root, tool.Source.Repo.Revision, target); err != nil {
		return "", nil, err
	}

	newEnv, err := runtime.Setup(ctx, m.runtimeDir, target, env)
	if err != nil {
		return "", nil, err
	}

	out, err := os.Create(doneFile + ".tmp")
	if err != nil {
		return "", nil, err
	}
	defer out.Close()

	if err := json.NewEncoder(out).Encode(newEnv); err != nil {
		return "", nil, err
	}

	return target, append(env, newEnv...), os.Rename(doneFile+".tmp", doneFile)
}

func (m *Manager) GetContext(ctx context.Context, tool types.Tool, cmd, env []string) (string, []string, error) {
	if tool.Source.Repo == nil {
		return tool.WorkingDir, env, nil
	}

	if tool.Source.Repo.VCS != "git" {
		return "", nil, fmt.Errorf("only git is supported, found VCS %s for %s", tool.Source.Repo.VCS, tool.ID)
	}

	for _, runtime := range m.runtimes {
		if runtime.Supports(cmd) {
			return m.setup(ctx, runtime, tool, env)
		}
	}

	return m.setup(ctx, &noopRuntime{}, tool, env)
}
