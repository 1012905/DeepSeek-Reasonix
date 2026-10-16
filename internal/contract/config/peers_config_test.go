package config

import (
	"slices"
	"testing"
)

func TestPeersAreOffUnlessTheUserFileTurnsThemOn(t *testing.T) {
	if Default().PeersEnabled() {
		t.Fatal("messages between sessions are on by default")
	}
	cfg, _ := loadScoped(t, "[peers]\nenabled = true\n", "")
	if !cfg.PeersEnabled() {
		t.Fatal("a user file's enabled = true was not honoured")
	}
}

// Turning this on opens a way for other sessions' text to reach this one, so it
// is the user's alone: a cloned repository can neither switch it on nor undo
// the user's choice.
func TestAProjectFileCannotSwitchPeersOn(t *testing.T) {
	cfg, _ := loadScoped(t, "", "[peers]\nenabled = true\n")
	if cfg.PeersEnabled() {
		t.Fatal("a cloned repo's reasonix.toml enabled messages between sessions")
	}
	if !slices.Contains(ignoredKeys(cfg), "peers.enabled") {
		t.Fatalf("ignored = %v, want peers.enabled reported", ignoredKeys(cfg))
	}
}

func TestAProjectFileCannotSwitchOffWhatTheUserChose(t *testing.T) {
	cfg, _ := loadScoped(t, "[peers]\nenabled = true\n", "[peers]\nenabled = false\n")
	if !cfg.PeersEnabled() {
		t.Fatal("the project file overrode the user's own peers setting")
	}
}
