package repos

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/locker"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/git"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Runtime interface {
	ID() string
	Supports(tool types.Tool, cmd []string) bool
	Binary(ctx context.Context, tool types.Tool, dataRoot, toolSource string, env []string) (bool, []string, error)
	Setup(ctx context.Context, tool types.Tool, dataRoot, toolSource string, env []string) ([]string, error)
	GetHash(tool types.Tool) (string, error)
}

type noopRuntime struct {
}

func (n noopRuntime) ID() string {
	return "none"
}

func (n noopRuntime) GetHash(_ types.Tool) (string, error) {
	return "", nil
}

func (n noopRuntime) Supports(_ types.Tool, _ []string) bool {
	return false
}

func (n noopRuntime) Binary(_ context.Context, _ types.Tool, _, _ string, _ []string) (bool, []string, error) {
	return false, nil, nil
}

func (n noopRuntime) Setup(_ context.Context, _ types.Tool, _, _ string, _ []string) ([]string, error) {
	return nil, nil
}

type Manager struct {
	cacheDir   string
	storageDir string
	gitDir     string
	runtimeDir string
	systemDirs []string
	runtimes   []Runtime
}

func New(cacheDir, systemDir string, runtimes ...Runtime) *Manager {
	var (
		systemDirs []string
		root       = filepath.Join(cacheDir, "repos")
	)

	if strings.TrimSpace(systemDir) != "" {
		systemDirs = regexp.MustCompile("[;:,]").Split(strings.TrimSpace(systemDir), -1)
	}

	return &Manager{
		cacheDir:   cacheDir,
		storageDir: root,
		gitDir:     filepath.Join(root, "git"),
		runtimeDir: filepath.Join(root, "runtimes"),
		systemDirs: systemDirs,
		runtimes:   runtimes,
	}
}

func (m *Manager) setup(ctx context.Context, runtime Runtime, tool types.Tool, env []string) (string, []string, error) {
	locker.Lock(tool.ID)
	defer locker.Unlock(tool.ID)

	runtimeHash, err := runtime.GetHash(tool)
	if err != nil {
		return "", nil, err
	}

	target := filepath.Join(m.storageDir, tool.Source.Repo.Revision, tool.Source.Repo.Path, tool.Source.Repo.Name, runtime.ID())
	targetFinal := filepath.Join(target, tool.Source.Repo.Path+runtimeHash)
	doneFile := targetFinal + ".done"
	envData, err := os.ReadFile(doneFile)
	if err == nil {
		var savedEnv []string
		if err := json.Unmarshal(envData, &savedEnv); err == nil {
			return targetFinal, append(env, savedEnv...), nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", nil, err
	}

	// Cleanup previous failed runs
	_ = os.RemoveAll(doneFile + ".tmp")
	_ = os.RemoveAll(doneFile)
	_ = os.RemoveAll(target)

	var (
		newEnv   []string
		isBinary bool
	)

	if isBinary, newEnv, err = runtime.Binary(ctx, tool, m.runtimeDir, targetFinal, env); err != nil {
		return "", nil, err
	} else if !isBinary {
		if tool.Source.Repo.VCS == "git" {
			if err := git.Checkout(ctx, m.gitDir, tool.Source.Repo.Root, tool.Source.Repo.Revision, target); err != nil {
				return "", nil, err
			}
		} else {
			if err := os.MkdirAll(target, 0755); err != nil {
				return "", nil, err
			}
		}

		newEnv, err = runtime.Setup(ctx, tool, m.runtimeDir, targetFinal, env)
		if err != nil {
			return "", nil, err
		}
	}

	out, err := os.Create(doneFile + ".tmp")
	if err != nil {
		return "", nil, err
	}
	defer out.Close()

	if err := json.NewEncoder(out).Encode(newEnv); err != nil {
		return "", nil, err
	}

	if err := out.Close(); err != nil {
		return "", nil, err
	}

	return targetFinal, append(env, newEnv...), os.Rename(doneFile+".tmp", doneFile)
}

func (m *Manager) GetContext(ctx context.Context, tool types.Tool, cmd, env []string) (string, []string, error) {
	for _, systemDir := range m.systemDirs {
		if strings.HasPrefix(tool.WorkingDir, systemDir) {
			return tool.WorkingDir, env, nil
		}
	}

	var isLocal bool
	if tool.Source.Repo == nil {
		isLocal = true
		d, _ := json.Marshal(tool)
		id := hash.Digest(d)[:12]
		tool.Source.Repo = &types.Repo{
			VCS:      "<local>",
			Root:     id,
			Path:     "/",
			Name:     id,
			Revision: id,
		}
	}

	for _, runtime := range m.runtimes {
		if runtime.Supports(tool, cmd) {
			log.Debugf("Runtime %s supports %v", runtime.ID(), cmd)
			wd, env, err := m.setup(ctx, runtime, tool, env)
			if isLocal {
				wd = tool.WorkingDir
			}
			return wd, env, err
		}
	}

	if isLocal {
		return tool.WorkingDir, env, nil
	}

	return m.setup(ctx, &noopRuntime{}, tool, env)
}
