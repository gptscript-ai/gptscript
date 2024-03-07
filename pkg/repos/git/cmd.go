package git

import (
	"context"
	"os/exec"

	"github.com/gptscript-ai/gptscript/pkg/debugcmd"
)

func newGitCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	debugcmd.SetupDebug(cmd)
	return cmd
}

func cloneBare(ctx context.Context, repo, toDir string) error {
	cmd := newGitCommand(ctx, "clone", "--bare", "--depth", "1", repo, toDir)
	return cmd.Run()
}

func gitWorktreeAdd(ctx context.Context, gitDir, commitDir, commit string) error {
	cmd := newGitCommand(ctx, "--git-dir", gitDir, "worktree", "add", "-f", commitDir, commit)
	return cmd.Run()
}

func fetchCommit(ctx context.Context, gitDir, commit string) error {
	cmd := newGitCommand(ctx, "--git-dir", gitDir, "fetch", "origin", commit)
	return cmd.Run()
}
