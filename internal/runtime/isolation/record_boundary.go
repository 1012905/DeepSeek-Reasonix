package isolation

import (
	"os"
	"path/filepath"
	"strings"

	"reasonix/internal/base/fileutil"
	"reasonix/internal/platform/gitcmd"
)

// bindRecord treats stored paths as claims, not authority. Only the actual
// marker slot and the store's opened repository can authorize reconstruction
// or the legacy sweep. Rejected records remain on disk, without Git mutation.
func (s *Store) bindRecord(mark string, d durableEntry) (durableEntry, bool) {
	slot := filepath.Dir(mark)
	if !regularRecord(mark) || !plainDirectory(slot) ||
		!sameExistingDirectory(filepath.Dir(slot), s.managedRoot) ||
		!filepath.IsAbs(d.Cand.WorktreeRoot) || !plainDirectory(d.Cand.WorktreeRoot) ||
		!sameExistingDirectory(filepath.Dir(d.Cand.WorktreeRoot), slot) {
		return d, false
	}
	source := s.repo.Top()
	if source.Verify() != nil || !sameExistingDirectory(d.Snap.RepoRoot, source.WorkTree) ||
		!sameExistingDirectory(d.Workspace, s.workspace) {
		return d, false
	}
	// The prefix binds both the original workspace and the candidate's tool root.
	prefix := filepath.FromSlash(d.Snap.Prefix)
	if filepath.IsAbs(prefix) || prefix == ".." || strings.HasPrefix(filepath.Clean(prefix), ".."+string(filepath.Separator)) ||
		!sameExistingDirectory(filepath.Join(source.WorkTree, prefix), s.workspace) ||
		!sameExistingDirectory(filepath.Join(d.Cand.WorktreeRoot, prefix), d.Cand.WorkspaceRoot) ||
		!containedDirectory(source.WorkTree, s.workspace) || !containedDirectory(d.Cand.WorktreeRoot, d.Cand.WorkspaceRoot) {
		return d, false
	}
	if d.Snap.Repo != (gitcmd.Repo{}) && !sameRepository(d.Snap.Repo, source) {
		return d, false
	}
	candidate, ok := linkedCandidateIdentity(d.Cand.WorktreeRoot, source)
	if !ok || d.Cand.Repo != (gitcmd.Repo{}) && !sameRepository(d.Cand.Repo, candidate) {
		return d, false
	}
	// Use checked identities; no persisted record is rewritten during discovery.
	d.Snap.Repo, d.Cand.Repo = source, candidate.Top()
	return d, true
}

// A linked worktree's marker and administrative backlink establish its identity
// without starting Git, including for old records without pinned Repo fields.
func linkedCandidateIdentity(root string, source gitcmd.Repo) (gitcmd.Repo, bool) {
	mark := filepath.Join(root, ".git")
	if !regularRecord(mark) {
		return gitcmd.Repo{}, false
	}
	raw, err := os.ReadFile(mark)
	if err != nil {
		return gitcmd.Repo{}, false
	}
	admin, ok := strings.CutPrefix(strings.TrimSpace(string(raw)), "gitdir: ")
	if !ok || admin == "" {
		return gitcmd.Repo{}, false
	}
	admin = filepath.FromSlash(admin)
	if !filepath.IsAbs(admin) {
		admin = filepath.Join(root, admin)
	}
	back, err := os.ReadFile(filepath.Join(admin, "gitdir"))
	if err != nil {
		return gitcmd.Repo{}, false
	}
	backPath := filepath.FromSlash(strings.TrimSpace(string(back)))
	if !filepath.IsAbs(backPath) {
		backPath = filepath.Join(admin, backPath)
	}
	// File identity alone also accepts hard-linked .git files in different
	// candidates. The backlink must name this directory, not merely its inode.
	if !sameExistingDirectory(filepath.Dir(backPath), root) {
		return gitcmd.Repo{}, false
	}
	repo := gitcmd.Repo{Dir: root, WorkTree: root, GitDir: admin, CommonDir: source.CommonDir}
	return repo, repo.Verify() == nil && !sameExistingDirectory(repo.GitDir, source.GitDir)
}

func sameRepository(a, b gitcmd.Repo) bool {
	return a.Verify() == nil && sameExistingDirectory(a.Dir, b.WorkTree) &&
		sameExistingDirectory(a.WorkTree, b.WorkTree) && sameExistingDirectory(a.GitDir, b.GitDir) &&
		sameExistingDirectory(a.CommonDir, b.CommonDir)
}

func sameExistingDirectory(a, b string) bool {
	left, err := os.Stat(a)
	if err != nil || !left.IsDir() {
		return false
	}
	right, err := os.Stat(b)
	return err == nil && right.IsDir() && os.SameFile(left, right)
}

func plainDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func regularRecord(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func containedDirectory(root, path string) bool {
	root, err := fileutil.ResolveExistingPath(root)
	if err != nil {
		return false
	}
	path, err = fileutil.ResolveExistingPath(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
