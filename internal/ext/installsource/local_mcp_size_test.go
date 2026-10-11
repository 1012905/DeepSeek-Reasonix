package installsource

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
)

// A local .mcp.json is held to the size a fetched one is, so the server cap
// does not stand alone against parse-time memory.
func TestLocalMCPJSONSizeBoundary(t *testing.T) {
	head := `{"mcpServers":{"counter":{"url":"https://example.com/mcp"}}}`
	for _, size := range []int{defaultFetchLimit, defaultFetchLimit + 1} {
		path := filepath.Join(testenv.TempDir(t), ".mcp.json")
		if err := os.WriteFile(path, []byte(head+strings.Repeat(" ", size-len(head))), 0o600); err != nil {
			t.Fatal(err)
		}
		entries, _, err := readMCPJSON(path)
		if size > defaultFetchLimit {
			if !errors.Is(err, ErrSourceUnreadable) || len(entries) != 0 {
				t.Fatalf("size %d: entries=%d err=%v, want ErrSourceUnreadable", size, len(entries), err)
			}
			continue
		}
		if err != nil || len(entries) != 1 {
			t.Fatalf("size %d: entries=%d err=%v, want the file read", size, len(entries), err)
		}
	}
}
