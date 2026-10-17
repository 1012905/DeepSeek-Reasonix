package serve

import (
	"path/filepath"
	"slices"
	"strings"

	"reasonix/internal/contract/config"
	"reasonix/internal/platform/worktree"
	"reasonix/internal/state/sessionstore"
)

type workspaceProject struct {
	root       string
	roots      []string
	remembered bool
}

func projectRoot(root string) string {
	return worktree.ProjectRoot(root, config.DeliveryWorktreeDir())
}

func projectIdentity(root string) string {
	return sessionstore.CanonicalSessionPath(projectRoot(root))
}

func groupWorkspaces(refs []rootRef) []workspaceProject {
	out := []workspaceProject{}
	index := map[string]int{}
	aliases := map[string]string{}
	for _, ref := range refs {
		if projectRoot(ref.dir) == ref.dir {
			aliases[projectIdentity(ref.dir)] = ref.dir
		}
	}
	for _, ref := range refs {
		root := projectRoot(ref.dir)
		identity := projectIdentity(root)
		if alias := aliases[identity]; alias != "" {
			root = alias
		}
		at, ok := index[identity]
		if !ok {
			at = len(out)
			index[identity] = at
			out = append(out, workspaceProject{root: root, roots: []string{root}})
		}
		p := &out[at]
		p.remembered = p.remembered || ref.remembered
		if !slices.Contains(p.roots, ref.dir) {
			p.roots = append(p.roots, ref.dir)
		}
	}
	return out
}

func rememberedProjects(paths []string) []workspaceProject {
	refs := make([]rootRef, 0, len(paths))
	for _, path := range paths {
		refs = append(refs, rootRef{dir: path, remembered: true})
	}
	return groupWorkspaces(refs)
}

func workspaceProjectOrder(paths []string) []string {
	out := []string{}
	for _, p := range rememberedProjects(paths) {
		out = append(out, projectIdentity(p.root))
	}
	return out
}

func hasWorkspaceProject(paths []string, dir string) bool {
	return slices.Contains(workspaceProjectOrder(paths), projectIdentity(dir))
}

func orderWorkspaceMembers(paths, order []string) []string {
	members := map[string][]string{}
	for _, path := range paths {
		root := projectIdentity(path)
		members[root] = append(members[root], path)
	}
	out := make([]string, 0, len(paths))
	for _, root := range order {
		out = append(out, members[root]...)
	}
	return out
}

func (h *Hub) sessionRoots() []rootRef {
	out := h.roots()
	for _, p := range groupWorkspaces(out) {
		if !slices.ContainsFunc(out, func(ref rootRef) bool { return ref.dir == p.root }) {
			out = append(out, rootRef{dir: p.root, remembered: p.remembered})
		}
	}
	return out
}

func (h *Hub) listedSessionRoot(path, root string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	dir := sessionstore.CanonicalSessionPath(filepath.Dir(path))
	for _, ref := range h.sessionRoots() {
		if dir == sessionstore.CanonicalSessionPath(SessionDirFor(ref.dir)) &&
			(strings.TrimSpace(root) == "" || projectIdentity(root) == projectIdentity(ref.dir)) {
			return ref.dir
		}
	}
	return ""
}
