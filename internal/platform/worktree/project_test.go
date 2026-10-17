package worktree

import (
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
)

func TestProjectRootUsesGitIdentityAndPreservesSelectedSubdirectory(t *testing.T) {
	requireGit(t)
	repo, managed := initRepo(t), testenv.TempDir(t)
	result, err := Create(t.Context(), opened(t, repo), managed)
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", filepath.Join("packages", "app")} {
		root := filepath.Join(result.WorkspaceRoot, suffix)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		if got, want := ProjectRoot(root, managed), filepath.Join(result.SourceRoot, suffix); got != want {
			t.Errorf("%s: project %s, want %s", root, got, want)
		}
	}
	if got := ProjectRoot(repo, managed); got != repo {
		t.Errorf("original directory moved: %s", got)
	}
	if got := ProjectRoot(result.WorkspaceRoot, testenv.TempDir(t)); got != result.WorkspaceRoot {
		t.Errorf("a separately managed checkout was folded: %s", got)
	}
	unknown := filepath.Join(managed, "same-name", filepath.Base(repo))
	if err := os.MkdirAll(unknown, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(unknown, managed); got != unknown {
		t.Errorf("a folder name was mistaken for a Git relationship: %s", got)
	}
	if err := os.Remove(filepath.Join(result.WorkspaceRoot, ".git")); err != nil {
		t.Fatal(err)
	}
	if got := ProjectRoot(result.WorkspaceRoot, managed); got != result.WorkspaceRoot {
		t.Errorf("an unreadable identity must retain its directory: %s", got)
	}
}
