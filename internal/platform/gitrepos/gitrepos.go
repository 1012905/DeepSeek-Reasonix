package gitrepos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"reasonix/internal/platform/gitcmd"
)

const (
	// MaxDepth is how far below the workspace a repository may sit: 1 is a
	// direct child, 2 a grandchild.
	MaxDepth = 2
	// MaxRepos bounds the repositories kept.
	MaxRepos = 24
	// maxDirs bounds the directories read while looking.
	maxDirs = 400
)

// Found is a repository inside a workspace. Dir is its path relative to the
// workspace, slash-separated; empty when the workspace is itself the repository.
type Found struct {
	Dir  string
	Repo gitcmd.Repo
}

// Result is what a search settled. Truncated is true when a bound ended the
// search before every directory was read, so the list may be incomplete.
type Result struct {
	Repos     []Found
	Truncated bool
}

// Discover lists the repositories of the workspace at root. A workspace that is
// a repository is just that one: nothing inside it is looked for. Otherwise the
// directories below it are read breadth first to MaxDepth, a directory that is
// the top of its own work tree is kept and not entered, and symbolic links are
// not followed, so nothing outside the workspace is read.
func Discover(ctx context.Context, root string) (Result, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	if repo, err := gitcmd.Open(ctx, abs); err == nil {
		return Result{Repos: []Found{{Repo: repo}}}, nil
	} else if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	var out Result
	type level struct {
		dir   string
		depth int
	}
	queue := []level{{abs, 0}}
	read := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= MaxDepth {
			continue
		}
		entries, err := os.ReadDir(cur.dir)
		if err != nil {
			continue
		}
		read++
		if read > maxDirs {
			out.Truncated = true
			break
		}
		for _, e := range entries {
			if !e.IsDir() || e.Type()&os.ModeSymlink != 0 {
				continue
			}
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			child := filepath.Join(cur.dir, e.Name())
			if repo, ok := topOfItsOwnTree(ctx, child); ok {
				if len(out.Repos) >= MaxRepos {
					out.Truncated = true
					continue
				}
				rel, _ := filepath.Rel(abs, child)
				out.Repos = append(out.Repos, Found{Dir: filepath.ToSlash(rel), Repo: repo})
				continue
			}
			queue = append(queue, level{child, cur.depth + 1})
		}
	}
	sort.Slice(out.Repos, func(i, j int) bool { return out.Repos[i].Dir < out.Repos[j].Dir })
	return out, nil
}

// topOfItsOwnTree opens dir and keeps it only when the work tree git resolved is
// dir itself: a folder inside some other repository resolves to that one and is
// not a repository of its own.
func topOfItsOwnTree(ctx context.Context, dir string) (gitcmd.Repo, bool) {
	if _, err := os.Lstat(filepath.Join(dir, ".git")); err != nil {
		return gitcmd.Repo{}, false
	}
	repo, err := gitcmd.Open(ctx, dir)
	if err != nil {
		return gitcmd.Repo{}, false
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil || filepath.Clean(real) != repo.WorkTree {
		return gitcmd.Repo{}, false
	}
	return repo, true
}

// ErrNotListed is returned for a repository name Discover would not have found:
// outside the workspace, deeper than MaxDepth, behind a link, not the top of
// its own work tree, or named when the workspace is itself a repository.
var ErrNotListed = errors.New("gitrepos: not a repository of this workspace")

// Resolve answers the repository a request named, by the same test Discover
// applies and never by trusting the text: dir is a slash-separated path below
// root, at most MaxDepth deep, with no link along it. An empty dir is the
// workspace's own repository, if it is one.
func Resolve(ctx context.Context, root, dir string) (gitcmd.Repo, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return gitcmd.Repo{}, err
	}
	if dir == "" {
		repo, err := gitcmd.Open(ctx, abs)
		if err != nil {
			return gitcmd.Repo{}, errors.Join(ErrNotListed, err)
		}
		return repo, nil
	}
	if _, err := gitcmd.Open(ctx, abs); err == nil {
		return gitcmd.Repo{}, ErrNotListed
	}
	parts := strings.Split(dir, "/")
	if len(parts) > MaxDepth || !filepath.IsLocal(filepath.FromSlash(dir)) {
		return gitcmd.Repo{}, ErrNotListed
	}
	at := abs
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return gitcmd.Repo{}, ErrNotListed
		}
		at = filepath.Join(at, p)
		info, err := os.Lstat(at)
		if err != nil || !info.IsDir() {
			return gitcmd.Repo{}, ErrNotListed
		}
	}
	repo, ok := topOfItsOwnTree(ctx, at)
	if !ok {
		return gitcmd.Repo{}, ErrNotListed
	}
	return repo, nil
}
