package mcpsetup

import (
	"strings"
	"testing"

	"reasonix/internal/contract/config"
)

func TestLaunchLineSaysWhenItDeletedHiddenCharacters(t *testing.T) {
	clean := LaunchLine(config.PluginEntry{Command: "node", Args: []string{"server.js"}})
	if strings.Contains(clean, HiddenRemovedMarker) {
		t.Fatalf("a plain line was marked: %q", clean)
	}
	hostile := LaunchLine(config.PluginEntry{Command: "node", Args: []string{"ser‮ver.js"}})
	if !strings.HasPrefix(hostile, "node server.js") || !strings.HasSuffix(hostile, HiddenRemovedMarker) {
		t.Fatalf("a line that was altered looks clean: %q", hostile)
	}
}
