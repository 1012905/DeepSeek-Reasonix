package isolation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"reasonix/internal/platform/worktree"
)

// entryFile sits beside a pending result's worktree. It names the result and
// is the only mark that makes a directory under the managed root ours to sweep.
const entryFile = "entry.json"

// durableEntry is what identifies a pending result. What it changed is not
// stored: the worktree is the result, and every read derives that from it.
type durableEntry struct {
	ID        string             `json:"id"`
	Workspace string             `json:"workspace"`
	Snap      worktree.Snapshot  `json:"snap"`
	Cand      worktree.Candidate `json:"cand"`
}

func entryPath(cand worktree.Candidate) string {
	return filepath.Join(filepath.Dir(cand.WorktreeRoot), entryFile)
}

func persist(e *Entry) error {
	raw, err := json.Marshal(durableEntry{ID: e.ID, Workspace: e.Workspace, Snap: e.Snap, Cand: e.Cand})
	if err != nil {
		return err
	}
	path := entryPath(e.Cand)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func forget(e *Entry) { _ = os.Remove(entryPath(e.Cand)) }

// durable lists the results on disk that were taken from this store's
// workspace. A result from another workspace is never offered here: applying
// it would write into that workspace from this one.
func (s *Store) durable() []durableEntry {
	managed, err := os.OpenRoot(s.managedRoot)
	if err != nil {
		return nil
	}
	defer managed.Close()
	entries, err := fs.ReadDir(managed.FS(), ".")
	if err != nil {
		return nil
	}
	var out []durableEntry
	for _, entry := range entries {
		if !entry.IsDir() || !recordDirectoryEntry(managed, entry.Name()) {
			continue
		}
		slot, err := managed.OpenRoot(entry.Name())
		if err != nil {
			continue
		}
		d, ok := s.readRecord(slot)
		slot.Close()
		if ok {
			out = append(out, d)
		}
	}
	return out
}

func (s *Store) readRecord(slot *os.Root) (durableEntry, bool) {
	if !recordRegularFile(slot, entryFile) {
		return durableEntry{}, false
	}
	raw, err := slot.ReadFile(entryFile)
	var d durableEntry
	if err != nil || json.Unmarshal(raw, &d) != nil || d.ID == "" {
		return d, false
	}
	return s.bindRecord(slot, d)
}

func (s *Store) durableIDs() []string {
	var ids []string
	for _, d := range s.durable() {
		ids = append(ids, d.ID)
	}
	return ids
}

// recover rebuilds a result from its worktree. Inspection errors retain the
// result for retry; a successfully inspected empty result is removed.
func (s *Store) recover(id string) (*Entry, error) {
	for _, d := range s.durable() {
		if d.ID != id {
			continue
		}
		ctx := context.Background()
		e := &Entry{ID: d.ID, Workspace: d.Workspace, Snap: d.Snap, Cand: d.Cand}
		tree, changes, err := worktree.CandidateTree(ctx, e.Snap, e.Cand)
		if err == nil && len(changes) > 0 {
			e.Stats, err = worktree.DiffStat(ctx, e.Snap, tree, changes)
		}
		if err != nil {
			return nil, fmt.Errorf("inspect isolated result %s; files retained: %w", id, err)
		}
		if len(changes) == 0 {
			s.remove(ctx, e)
			return nil, nil
		}
		e.Tree, e.Changes = tree, changes
		s.mu.Lock()
		s.entries[e.ID] = e
		s.mu.Unlock()
		return e, nil
	}
	return nil, nil
}

// SweepExpired removes this workspace's results nobody settled within maxAge.
// Only directories carrying an entry mark are touched.
func (s *Store) SweepExpired(ctx context.Context, maxAge time.Duration) {
	if s == nil {
		return
	}
	for _, d := range s.durable() {
		info, err := os.Stat(entryPath(d.Cand))
		if err != nil || time.Since(info.ModTime()) < maxAge {
			continue
		}
		s.remove(ctx, &Entry{ID: d.ID, Snap: d.Snap, Cand: d.Cand})
	}
}
