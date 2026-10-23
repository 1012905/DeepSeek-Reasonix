package agent

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"reasonix/internal/base/hostwrite"
	"reasonix/internal/runtime/writeclaim"
	"reasonix/internal/state/sessiontemp"
)

// outsideObservedWrites names the files this turn wrote where nothing the host
// watches can speak for them. A write there is neither a probe nor the work
// product: the host saw the call succeed and can say nothing about what it left.
func (a *Agent) outsideObservedWrites() []string {
	if a == nil || a.task.ledger == nil || a.writeWorkspaceRoot == "" {
		return nil
	}
	var outside []string
	for _, r := range a.task.ledger.Receipts() {
		if !r.Success || !(r.Write || r.Mutation) {
			continue
		}
		for _, p := range r.Paths {
			if p == "" || slices.Contains(r.HostOwned, p) || a.pathInWorkspace(p) || underScratchOrCache(p) {
				continue
			}
			if !slices.Contains(outside, p) {
				outside = append(outside, p)
			}
		}
	}
	return a.relativeUnique(outside)
}

// underScratchOrCache reports a path the host owns and disposes of: inside the
// session's own scratch directory, or inside a toolchain cache. A public
// temporary root is not exempt on its own — a person who asked for a file there
// asked for something to keep, and nothing the host watches says it arrived.
func underScratchOrCache(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	target := filepath.Clean(path)
	if withoutSessionScratch(target) != target {
		return true
	}
	for _, dir := range hostwrite.Dirs() {
		dir = strings.TrimSpace(dir)
		if dir == "" || hostTempRoot(dir) {
			continue
		}
		clean := filepath.Clean(dir)
		if !writeclaim.PathWithin(clean, target) {
			continue
		}
		// An executable installed under a cache is the user's, not the cache's:
		// `go install` lands in ~/go/bin, which is what the turn was asked for.
		if rel, err := filepath.Rel(clean, target); err == nil && strings.HasPrefix(filepath.ToSlash(rel), "bin/") {
			return false
		}
		return true
	}
	return false
}

// withoutSessionScratch drops the prefix of path through the session's scratch
// directory, or returns path unchanged when it does not lie inside one. A
// directory earns that name only as a direct child of a root the host uses for
// temporary files: any directory could otherwise claim the name.
func withoutSessionScratch(path string) string {
	prefix := sessiontemp.Prefix()
	for dir := path; ; dir = filepath.Dir(dir) {
		if strings.HasPrefix(filepath.Base(dir), prefix) && hostTempRoot(filepath.Dir(dir)) {
			return filepath.Dir(dir)
		}
		if parent := filepath.Dir(dir); parent == dir {
			return path
		}
	}
}

// hostTempRoot reports a directory that is merely where temporary files go.
func hostTempRoot(dir string) bool {
	clean := filepath.Clean(dir)
	if temp := os.TempDir(); temp != "" && clean == filepath.Clean(temp) {
		return true
	}
	switch clean {
	case "/tmp", "/private/tmp", "/private/var/folders":
		return true
	}
	return false
}
