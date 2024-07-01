package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/locker"
	"github.com/gptscript-ai/gptscript/pkg/config"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/loader/github"
	"github.com/gptscript-ai/gptscript/pkg/repos/git"
	"github.com/gptscript-ai/gptscript/pkg/repos/runtimes/golang"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

const credentialHelpersRepo = "github.com/gptscript-ai/gptscript-credential-helpers"

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
	storageDir       string
	gitDir           string
	runtimeDir       string
	credHelperDirs   credentials.CredentialHelperDirs
	runtimes         []Runtime
	credHelperConfig *credHelperConfig
}

type credHelperConfig struct {
	lock        sync.Mutex
	initialized bool
	cliCfg      *config.CLIConfig
	env         []string
}

func New(cacheDir string, runtimes ...Runtime) *Manager {
	root := filepath.Join(cacheDir, "repos")
	return &Manager{
		storageDir:     root,
		gitDir:         filepath.Join(root, "git"),
		runtimeDir:     filepath.Join(root, "runtimes"),
		credHelperDirs: credentials.GetCredentialHelperDirs(cacheDir),
		runtimes:       runtimes,
	}
}

func (m *Manager) EnsureCredentialHelpers(ctx context.Context) error {
	if m.credHelperConfig == nil {
		return nil
	}
	m.credHelperConfig.lock.Lock()
	defer m.credHelperConfig.lock.Unlock()

	if !m.credHelperConfig.initialized {
		if err := m.deferredSetUpCredentialHelpers(ctx, m.credHelperConfig.cliCfg, m.credHelperConfig.env); err != nil {
			return err
		}
		m.credHelperConfig.initialized = true
	}

	return nil
}

func (m *Manager) SetUpCredentialHelpers(_ context.Context, cliCfg *config.CLIConfig, env []string) error {
	m.credHelperConfig = &credHelperConfig{
		cliCfg: cliCfg,
		env:    env,
	}
	return nil
}

func (m *Manager) deferredSetUpCredentialHelpers(ctx context.Context, cliCfg *config.CLIConfig, env []string) error {
	var (
		helperName = cliCfg.CredentialsStore
		suffix     string
	)
	if helperName == "wincred" {
		suffix = ".exe"
	}

	// The file helper is built-in and does not need to be compiled.
	if helperName == "file" {
		return nil
	}

	locker.Lock("gptscript-credential-helpers")
	defer locker.Unlock("gptscript-credential-helpers")

	// Load the last-checked file to make sure we haven't checked the repo in the last 24 hours.
	now := time.Now()
	lastChecked, err := os.ReadFile(m.credHelperDirs.LastCheckedFile)
	if err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(lastChecked))); err == nil && now.Sub(t) < 24*time.Hour {
			// Make sure the binary still exists, and if it does, return.
			if _, err := os.Stat(filepath.Join(m.credHelperDirs.BinDir, "gptscript-credential-"+helperName+suffix)); err == nil {
				log.Debugf("Credential helper %s up-to-date as of %v, checking for updates after %v", helperName, t, t.Add(24*time.Hour))
				return nil
			}
		}
	}

	// Load the credential helpers repo information.
	_, _, repo, _, err := github.Load(ctx, nil, credentialHelpersRepo)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Dir(m.credHelperDirs.LastCheckedFile), 0755); err != nil {
		return err
	}

	// Update the last-checked file.
	if err := os.WriteFile(m.credHelperDirs.LastCheckedFile, []byte(now.Format(time.RFC3339)), 0644); err != nil {
		return err
	}

	var needsBuild bool

	// Check the last revision shasum and see if it is different from the current one.
	lastRevision, err := os.ReadFile(m.credHelperDirs.RevisionFile)
	if (err == nil && strings.TrimSpace(string(lastRevision)) != repo.Revision) || errors.Is(err, fs.ErrNotExist) {
		// Need to pull the latest version.
		needsBuild = true
		if err := git.Checkout(ctx, m.gitDir, repo.Root, repo.Revision, filepath.Join(m.credHelperDirs.RepoDir, repo.Revision)); err != nil {
			return err
		}
		// Update the revision file to the new revision.
		if err := os.WriteFile(m.credHelperDirs.RevisionFile, []byte(repo.Revision), 0644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if !needsBuild {
		// Check for the existence of the gptscript-credential-osxkeychain binary.
		// If it's there, we have no need to build it and can just return.
		if _, err := os.Stat(filepath.Join(m.credHelperDirs.BinDir, "gptscript-credential-"+helperName+suffix)); err == nil {
			return nil
		}
	}

	// Find the Go runtime and use it to build the credential helper.
	for _, runtime := range m.runtimes {
		if strings.HasPrefix(runtime.ID(), "go") {
			goRuntime := runtime.(*golang.Runtime)
			return goRuntime.BuildCredentialHelper(ctx, helperName, m.credHelperDirs, m.runtimeDir, repo.Revision, env)
		}
	}

	return fmt.Errorf("no Go runtime found to build the credential helper")
}

func (m *Manager) setup(ctx context.Context, runtime Runtime, tool types.Tool, env []string) (string, []string, error) {
	locker.Lock(tool.ID)
	defer locker.Unlock(tool.ID)

	target := filepath.Join(m.storageDir, tool.Source.Repo.Revision, tool.Source.Repo.Path, tool.Source.Repo.Name, runtime.ID())
	targetFinal := filepath.Join(target, tool.Source.Repo.Path)
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

	if err := git.Checkout(ctx, m.gitDir, tool.Source.Repo.Root, tool.Source.Repo.Revision, target); err != nil {
		return "", nil, err
	}

	newEnv, err := runtime.Setup(ctx, m.runtimeDir, targetFinal, env)
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

	if err := out.Close(); err != nil {
		return "", nil, err
	}

	return targetFinal, append(env, newEnv...), os.Rename(doneFile+".tmp", doneFile)
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
			log.Debugf("Runtime %s supports %v", runtime.ID(), cmd)
			return m.setup(ctx, runtime, tool, env)
		}
	}

	return m.setup(ctx, &noopRuntime{}, tool, env)
}
