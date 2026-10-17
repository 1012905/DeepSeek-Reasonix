package worktree

import (
	"os"
	"path/filepath"

	"reasonix/internal/base/workspaceid"
)

// ProjectRoot maps a managed checkout to its main checkout's corresponding
// folder. Unknown or missing Git identities keep their own directory.
func ProjectRoot(root, managedRoot string) string {
	if !IsManagedPath(root, managedRoot) {
		return root
	}
	info := workspaceid.Describe(root)
	if info.RepoDir == "" || info.RepoDir == info.GitDir || filepath.Base(info.RepoDir) != ".git" {
		return root
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			rel, err := filepath.Rel(dir, abs)
			if err != nil {
				return root
			}
			return filepath.Join(filepath.Dir(info.RepoDir), rel)
		}
		if filepath.Dir(dir) == dir {
			return root
		}
	}
}
