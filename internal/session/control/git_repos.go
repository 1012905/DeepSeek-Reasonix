package control

import (
	"context"

	"reasonix/internal/platform/gitcmd"
	"reasonix/internal/platform/gitrepos"
)

// GitRepos lists the repositories of the workspace: the workspace itself when it
// is one, otherwise the repositories found below it within the search bounds.
func (c *Controller) GitRepos(ctx context.Context) (gitrepos.Result, error) {
	return gitrepos.Discover(ctx, c.WorkspaceRoot())
}

// RepoFor is the repository a request named by its directory below the workspace,
// as GitRepos lists it. An empty dir is the workspace's own repository, the one
// every git read answered for before a workspace could hold several. A name
// GitRepos would not list is gitrepos.ErrNotListed.
func (c *Controller) RepoFor(ctx context.Context, dir string) (gitcmd.Repo, error) {
	if dir == "" {
		return c.WorkspaceRepo(), nil
	}
	return gitrepos.Resolve(ctx, c.WorkspaceRoot(), dir)
}
