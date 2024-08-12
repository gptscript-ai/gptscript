package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

var (
	gitCheck    sync.Once
	externalGit bool
)

func usePureGo() bool {
	if os.Getenv("GPTSCRIPT_PURE_GO_GIT") == "true" {
		return true
	}
	gitCheck.Do(func() {
		_, err := exec.LookPath("git")
		externalGit = err == nil
	})
	return !externalGit
}

func lsRemotePureGo(_ context.Context, repo, ref string) (string, error) {
	// Clone the repository in memory
	r := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo},
	})

	refs, err := r.List(&git.ListOptions{
		PeelingOption: git.AppendPeeled,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list remote refs: %w", err)
	}

	for _, checkRef := range refs {
		if checkRef.Name().Short() == ref {
			return checkRef.Hash().String(), nil
		}
	}

	return "", fmt.Errorf("failed to find remote ref %q", ref)
}

func checkoutPureGo(ctx context.Context, _, repo, commit, toDir string) error {
	log.InfofCtx(ctx, "Checking out %s to %s", commit, toDir)
	// Clone the repository
	r, err := git.PlainCloneContext(ctx, toDir, false, &git.CloneOptions{
		URL:        repo,
		NoCheckout: true,
	})
	if err != nil {
		return fmt.Errorf("failed to clone the repo: %w", err)
	}

	// Fetch the specific commit
	err = r.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+%s:%s", commit, commit)),
		},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch the commit: %w", err)
	}

	// Checkout the specific commit
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commit),
	})
	if err != nil {
		return fmt.Errorf("failed to checkout the commit: %w", err)
	}

	return nil
}
