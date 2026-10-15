package serve

import (
	"path/filepath"
	"strings"

	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/store"
)

// sessionPage is the newest rows of one folder plus the counts of the whole.
type sessionPage struct {
	Sessions                   []treeSession
	Total, Archived, Remaining int
	// Said is whether any conversation here has had a turn, or a first turn is
	// in flight: what separates a folder with history from one only opened.
	Said bool
}

// workspacePage lists one folder's conversations, hiding what the pickers
// hide: subagent traces and redundant recovery copies. A copy that a pane has
// open stays, because something is driving it. limit bounds the rows built (0
// is all of them); the counts always cover every row the folder lists.
func (h *Hub) workspacePage(root, name string, open map[string]string, window treeWindow, limit int) sessionPage {
	dir := SessionDirFor(root)
	listed, err := sessionstore.ListSessions(dir)
	if err != nil {
		return sessionPage{Sessions: []treeSession{}}
	}
	titles := h.titleCacheFor(dir).snapshot()
	heldBy := openLookup(open)
	byID := make(map[string]sessionstore.SessionInfo, len(listed))
	for _, si := range listed {
		byID[sessionstore.BranchID(si.Path)] = si
	}
	out := make([]treeSession, 0, min(len(listed), max(limit, 0)))
	var total, archived, remaining, windowed int
	workspaceNamed := window.query != "" && strings.Contains(strings.ToLower(name), window.query)
	// Lineage root -> the row that copies of it fold into, or -1 when that row
	// is past the window. The list is newest-first, so the first copy to arrive
	// is the one still being written.
	lead := map[string]int{}
	for _, si := range listed {
		base := filepath.Base(si.Path)
		if store.IsSubagentTranscriptName(base) {
			continue
		}
		runtimeID := heldBy(si.Path)
		if runtimeID == "" && hiddenRecoveryCopy(si, "") {
			continue
		}
		name := strings.TrimSuffix(base, ".jsonl")
		// A pane driving a copy keeps its own row: folding it away would hide
		// the conversation someone is looking at.
		folds := si.Recovered && runtimeID == ""
		if folds {
			if at, ok := lead[recoveryLineageRoot(si, byID)]; ok {
				if at >= 0 {
					out[at].Copies = append(out[at].Copies, treeSession{Path: si.Path, Name: name, Turns: si.Turns})
				}
				continue
			}
		}
		total++
		if si.Archived {
			archived++
		}
		titleOf := rowTitle(titles, si, name)
		matches := window.matchesRow(si, titleOf, name, workspaceNamed)
		inWindow := matches && (limit == 0 || windowed < limit)
		if inWindow {
			windowed++
		}
		if !inWindow && !(si.Pinned && !si.Archived) {
			if matches {
				remaining++
			}
			if folds {
				lead[recoveryLineageRoot(si, byID)] = -1
			}
			continue
		}
		title := titleOf()
		if folds {
			lead[recoveryLineageRoot(si, byID)] = len(out)
		}
		out = append(out, treeSession{
			Path: si.Path, Name: name, Title: title, Turns: si.Turns, RuntimeID: runtimeID, Archived: si.Archived, Unread: si.Unread, LastActiveAt: activeAtMs(si), Legacy: sessionstore.IsForeignSessionLog(si.Path),
		})
		decorateArchiveState(&out[len(out)-1], si.Path)
	}
	attachVersions(dir, out, h.openSessionsIn(root))
	unlisted := h.unlistedOpenSessions(root, out)
	// Never nil: the frontend iterates rows, and a null blanks the window.
	rows := append(append([]treeSession{}, unlisted...), out...)
	said := len(unlisted) > 0
	for _, si := range listed {
		said = said || si.Turns > 0
	}
	return sessionPage{Sessions: rows, Total: total + len(unlisted), Archived: archived, Remaining: remaining, Said: said}
}

// openLookup answers which pane holds a listed path. Canonicalising a path costs
// a stat per ancestor, so it is only paid for a file name some pane holds; on a
// folder of thousands of conversations that is the difference between one
// request and a second of disk.
func openLookup(open map[string]string) func(path string) string {
	names := make(map[string]bool, len(open))
	for held := range open {
		names[filepath.Base(held)] = true
	}
	return func(path string) string {
		if !names[filepath.Base(path)] {
			return ""
		}
		return open[sessionstore.CanonicalSessionPath(path)]
	}
}

// rowTitle reads a row's title once, on first use: a name the user typed
// outranks the generated one, and without that a rename would be written to the
// sidecar and never show up.
func rowTitle(titles titleSnapshot, si sessionstore.SessionInfo, name string) func() string {
	var title string
	done := false
	return func() string {
		if !done {
			done = true
			title = titles.display(name+".jsonl", titleSource(si.Preview), sessionstore.SessionContentModTime(si.Path).UnixNano(), si.CustomTitle)
			if title == "" {
				title = previewTitle(si.Preview)
			}
		}
		return title
	}
}

// matchesRow says whether a row is one the window's scope and word are about.
// The word is read against the title, or the name where there is none, and a
// folder whose own name holds it keeps every row.
func (w treeWindow) matchesRow(si sessionstore.SessionInfo, title func() string, name string, workspaceNamed bool) bool {
	if (w.scope == scopeArchived && !si.Archived) || (w.scope == scopeActive && si.Archived) {
		return false
	}
	if w.query == "" || workspaceNamed {
		return true
	}
	label := title()
	if label == "" {
		label = name
	}
	return strings.Contains(strings.ToLower(label), w.query)
}

func activeAtMs(si sessionstore.SessionInfo) int64 {
	if si.LastActivityAt.IsZero() {
		return 0
	}
	return si.LastActivityAt.UnixMilli()
}
