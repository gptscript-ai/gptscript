package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/locker"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/repos/git"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/golang"
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
	cacheDir         string
	storageDir       string
	gitDir           string
	runtimeDir       string
	runtimes         []Runtime
	credHelperConfig *credHelperConfig
}

type credHelperConfig struct {
	lock        sync.Mutex
	initialized bool
	cliCfg      *config.CLIConfig
}

func New(cacheDir string, runtimes ...Runtime) *Manager {
	root := filepath.Join(cacheDir, "repos")
	return &Manager{
		cacheDir:   cacheDir,
		storageDir: root,
		gitDir:     filepath.Join(root, "git"),
		runtimeDir: filepath.Join(root, "runtimes"),
		runtimes:   runtimes,
	}
}

func (m *Manager) EnsureCredentialHelpers(ctx context.Context) error {
	if m.credHelperConfig == nil {
		return nil
	}
	m.credHelperConfig.lock.Lock()
	defer m.credHelperConfig.lock.Unlock()

	if !m.credHelperConfig.initialized {
		if err := m.deferredSetUpCredentialHelpers(ctx, m.credHelperConfig.cliCfg); err != nil {
			return err
		}
		m.credHelperConfig.initialized = true
	}

	return nil
}

func (m *Manager) SetUpCredentialHelpers(_ context.Context, cliCfg *config.CLIConfig) error {
	m.credHelperConfig = &credHelperConfig{
		cliCfg: cliCfg,
	}
	return nil
}

func (m *Manager) deferredSetUpCredentialHelpers(ctx context.Context, cliCfg *config.CLIConfig) error {
	var (
		helperName       = cliCfg.CredentialsStore
		distInfo, suffix string
	)
	// The file helper is built-in and does not need to be downloaded.
	if helperName == "file" {
		return nil
	}
	switch helperName {
	case "wincred":
		suffix = ".exe"
	default:
		distInfo = fmt.Sprintf("-%s-%s", runtime.GOOS, runtime.GOARCH)
	}

	repoName := credentials.RepoNameForCredentialStore(helperName)

	locker.Lock(repoName)
	defer locker.Unlock(repoName)

	credHelperDirs := credentials.GetCredentialHelperDirs(m.cacheDir, helperName)

	// Load the last-checked file to make sure we haven't checked the repo in the last 24 hours.
	now := time.Now()
	lastChecked, err := os.ReadFile(credHelperDirs.LastCheckedFile)
	if err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(lastChecked))); err == nil && now.Sub(t) < 24*time.Hour {
			// Make sure the binary still exists, and if it does, return.
			if _, err := os.Stat(filepath.Join(credHelperDirs.BinDir, "gptscript-credential-"+helperName+suffix)); err == nil {
				log.Debugf("Credential helper %s up-to-date as of %v, checking for updates after %v", helperName, t, t.Add(24*time.Hour))
				return nil
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(credHelperDirs.LastCheckedFile), 0755); err != nil {
		return err
	}

	// Update the last-checked file.
	if err := os.WriteFile(credHelperDirs.LastCheckedFile, []byte(now.Format(time.RFC3339)), 0644); err != nil {
		return err
	}

	gitURL, err := credentials.GitURLForRepoName(repoName)
	if err != nil {
		return err
	}

	tool := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name: repoName,
			},
		},
		Source: types.ToolSource{
			Repo: &types.Repo{
				Root: gitURL,
			},
		},
	}
	tag, err := golang.GetLatestTag(tool)
	if err != nil {
		return err
	}

	var needsDownloaded bool
	// Check the last revision shasum and see if it is different from the current one.
	lastRevision, err := os.ReadFile(credHelperDirs.RevisionFile)
	if (err == nil && strings.TrimSpace(string(lastRevision)) != tool.Source.Repo.Root+tag) || errors.Is(err, fs.ErrNotExist) {
		// Need to pull the latest version.
		needsDownloaded = true
		// Update the revision file to the new revision.
		if err = os.WriteFile(credHelperDirs.RevisionFile, []byte(tool.Source.Repo.Root+tag), 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if !needsDownloaded {
		// Check for the existence of the credential helper binary.
		// If it's there, we have no need to download it and can just return.
		if _, err = os.Stat(filepath.Join(credHelperDirs.BinDir, "gptscript-credential-"+helperName+suffix)); err == nil {
			return nil
		}
	}

	// Find the Go runtime and use it to build the credential helper.
	for _, rt := range m.runtimes {
		if strings.HasPrefix(rt.ID(), "go") {
			return rt.(*golang.Runtime).DownloadCredentialHelper(ctx, tool, helperName, distInfo, suffix, credHelperDirs.BinDir)
		}
	}

	return fmt.Errorf("no Go runtime found to build the credential helper")
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
