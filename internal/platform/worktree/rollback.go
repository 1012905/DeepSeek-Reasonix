package worktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"reasonix/internal/platform/gitcmd"
)

// DiscardUnpublished rolls back a fresh worktree before any session can use it.
// Published worktrees are durable and must never be passed here.
func DiscardUnpublished(ctx context.Context, repo gitcmd.Repo, result Result) error {
	registered, _, err := runGit(ctx, repo, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return err
	}
	root := result.WorktreeRoot
	if parent, err := filepath.EvalSymlinks(filepath.Dir(root)); err == nil {
		root = filepath.Join(parent, filepath.Base(root))
	}
	found := false
	for entry := range strings.SplitSeq(registered, "\x00") {
		entry = filepath.ToSlash(entry)
		if entry == "worktree "+filepath.ToSlash(root) || entry == "worktree "+filepath.ToSlash(result.WorktreeRoot) {
			found = true
		}
	}
	if !found {
		_, _, branchErr := runGit(ctx, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+result.Branch)
		var exit *exec.ExitError
		if errors.As(branchErr, &exit) && exit.ExitCode() == 1 {
			if _, statErr := os.Lstat(result.WorktreeRoot); errors.Is(statErr, os.ErrNotExist) {
				return nil
			}
		}
		return fmt.Errorf("unregistered partial worktree retained: %s", result.WorktreeRoot)
	}
	head, _, err := runGit(ctx, repo, "rev-parse", "--verify", "refs/heads/"+result.Branch)
	if err != nil || strings.TrimSpace(head) != result.Head {
		return fmt.Errorf("unpublished worktree branch changed; retained %s", result.WorktreeRoot)
	}
	if _, _, err := runGit(ctx, repo, "worktree", "remove", "--force", result.WorktreeRoot); err != nil {
		return err
	}
	_, _, err = runGit(ctx, repo, "update-ref", "-d", "refs/heads/"+result.Branch, result.Head)
	return err
}
