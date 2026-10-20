package migration

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"strings"

	"reasonix/internal/contract/config"
)

const desktopRegistryPath = "desktop/workspace-state-v1.json"

// desktopSessionOwners reads which workspace root 1.x Desktop filed each
// session under; the by-id store itself carries no workspace. An unreadable
// registry yields no owners, and those sessions fall back to the chosen workspace.
func desktopSessionOwners(tree fs.FS, home string) map[string]string {
	owners, _, _ := readDesktopRegistry(tree, home)
	return owners
}

// desktopArchivedSessions reads which conversations 1.x Desktop had filed away.
// The archive list lived in the shell rather than the session store, so without it
// every archived conversation comes back as an active one after the move.
func desktopArchivedSessions(tree fs.FS, home string) map[string]bool {
	_, archived, _ := readDesktopRegistry(tree, home)
	return archived
}

// readDesktopRegistry reads both halves of the 1.x registry in one pass: which
// workspace owns each conversation, and which ones the person had archived.
func readDesktopRegistry(tree fs.FS, home string) (map[string]string, map[string]bool, error) {
	raw, err := fs.ReadFile(tree, path.Join(home, desktopRegistryPath))
	if err != nil {
		return nil, nil, err
	}
	var state struct {
		Workspaces map[string]struct {
			Root       string   `json:"root"`
			SessionIDs []string `json:"sessionIds"`
		} `json:"workspaces"`
		ArchivedSessionIDs []string `json:"archivedSessionIds"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, nil, err
	}
	owners := map[string]string{}
	for _, ws := range state.Workspaces {
		for _, id := range ws.SessionIDs {
			owners[id] = ws.Root
		}
	}
	archived := map[string]bool{}
	for _, id := range state.ArchivedSessionIDs {
		if id = strings.TrimSpace(id); id != "" {
			archived[id] = true
		}
	}
	return owners, archived, nil
}

// routeToOwner sends a session to its former workspace's session directory
// while that workspace still exists, and otherwise declines.
func routeToOwner(owners map[string]string, note func(root string)) func(string) string {
	if len(owners) == 0 {
		return nil
	}
	return func(id string) string {
		root := strings.TrimSpace(owners[id])
		if root == "" {
			return ""
		}
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			return ""
		}
		note(root)
		return config.ProjectSessionDir(root)
	}
}
