package sessionstore

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"reasonix/internal/state/store"
)

// ListSessionOrder returns every *.jsonl session under dir in the same
// most-recently-active order used by ListSessions, from file metadata and
// branch sidecars. A session whose files are unchanged since the last listing
// is answered from the remembered index (listindex.go). A missing directory is
// not an error.
func ListSessionOrder(dir string) ([]SessionOrderInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	stamps := newStampSet(entries)
	index := loadListIndex(dir)
	next := make(map[string]listIndexEntry, len(entries))
	var out []SessionOrderInfo
	reused := 0
	for _, e := range entries {
		if e.IsDir() || !store.IsSessionTranscriptName(e.Name()) {
			continue
		}
		full := filepath.Join(dir, e.Name())
		print, ok := stamps.of(e, full)
		if !ok {
			continue
		}
		if was, hit := index.Entries[e.Name()]; hit && was.Print == print {
			reused++
			next[e.Name()] = was
			if !was.Skip {
				info := was.Info
				info.Path = full
				out = append(out, info)
			}
			continue
		}
		info, skip := orderInfoOf(full, e)
		next[e.Name()] = listIndexEntry{Print: print, Skip: skip, Info: info}
		if !skip {
			out = append(out, info)
		}
	}
	saveListIndex(dir, index, next, reused)
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastActivityAt.Equal(out[j].LastActivityAt) {
			return out[i].Path < out[j].Path
		}
		return out[i].LastActivityAt.After(out[j].LastActivityAt)
	})
	return out, nil
}

// orderInfoOf reads what one session's listing row needs from its own files.
// skip is a session no list shows: one a branch superseded.
func orderInfoOf(full string, e fs.DirEntry) (SessionOrderInfo, bool) {
	info, err := e.Info()
	if err != nil {
		return SessionOrderInfo{}, true
	}
	contentMod := SessionContentModTime(full)
	if contentMod.IsZero() {
		contentMod = info.ModTime()
	}
	row := SessionOrderInfo{Path: full, CreatedAt: info.ModTime(), LastActivityAt: contentMod, ModTime: contentMod, Scope: "global"}
	meta, ok, err := LoadBranchMeta(full)
	if err != nil || !ok {
		return row, false
	}
	if meta.Superseded {
		return SessionOrderInfo{}, true
	}
	if !meta.CreatedAt.IsZero() {
		row.CreatedAt = meta.CreatedAt
	}
	if !meta.UpdatedAt.IsZero() {
		row.LastActivityAt = meta.UpdatedAt
		row.ModTime = meta.UpdatedAt
	}
	row.Scope = meta.DefaultScope()
	row.WorkspaceRoot = meta.WorkspaceRoot
	row.TopicID = meta.TopicID
	row.TopicTitle = meta.TopicTitle
	row.CustomTitle = meta.CustomTitle
	row.Archived = meta.Archived
	row.Pinned = meta.Pinned
	row.Unread = meta.Unread()
	row.Recovered = meta.Recovered
	row.RecoveryReason = meta.RecoveryReason
	row.RecoveryDigest = meta.RecoveryDigest
	row.ParentID = meta.ParentID
	row.RecoveryRootID = meta.RecoveryRootID
	row.Turns = meta.Turns
	row.Preview = meta.Preview
	row.SchemaVersion = meta.SchemaVersion
	row.Revision = meta.Revision
	row.ContentDigest = meta.ContentDigest
	return row, false
}
