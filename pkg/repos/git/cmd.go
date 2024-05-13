package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/gptscript-ai/gptscript/pkg/debugcmd"
)

func newGitCommand(ctx context.Context, args ...string) *debugcmd.WrappedCmd {
	if log.IsDebug() {
		log.Debugf("running git command: %s", strings.Join(args, " "))
	}
	cmd := debugcmd.New(ctx, "git", args...)
	return cmd
}

func LsRemote(ctx context.Context, repo, ref string) (string, error) {
	cmd := newGitCommand(ctx, "ls-remote", repo, ref)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	for _, line := range strings.Split(cmd.Stdout(), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == ref {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("failed to find remote %q as %q", repo, ref)
}

func cloneBare(ctx context.Context, repo, toDir string) error {
	cmd := newGitCommand(ctx, "clone", "--bare", "--depth", "1", repo, toDir)
	return cmd.Run()
}

func gitWorktreeAdd(ctx context.Context, gitDir, commitDir, commit string) error {
	// The double -f is intentional
	cmd := newGitCommand(ctx, "--git-dir", gitDir, "worktree", "add", "-f", "-f", commitDir, commit)
	return cmd.Run()
}

func fetchCommit(ctx context.Context, gitDir, commit string) error {
	cmd := newGitCommand(ctx, "--git-dir", gitDir, "fetch", "origin", commit)
	return cmd.Run()
}
