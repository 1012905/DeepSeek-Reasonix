package config

import (
	"slices"
	"testing"
)

// On Windows the bash sandbox is the user's explicit choice: an older config's
// "enforce" does not switch it on, and only windows_integrity does.
func TestWindowsEnforcesOnlyWhenWindowsIntegrityIsOn(t *testing.T) {
	cfg := Default()
	if got := cfg.BashModeForGOOS("windows"); got != "off" {
		t.Fatalf("default on Windows = %q, want off", got)
	}
	cfg.Sandbox.Bash = "enforce"
	if got := cfg.BashModeForGOOS("windows"); got != "off" {
		t.Fatalf(`an explicit bash = "enforce" turned the Windows sandbox on: %q`, got)
	}
	cfg.Sandbox.WindowsIntegrity = true
	if got := cfg.BashModeForGOOS("windows"); got != "enforce" {
		t.Fatalf("windows_integrity = true gave %q, want enforce", got)
	}
	if got := cfg.BashModeForGOOS("linux"); got != "enforce" {
		t.Fatalf("Linux is unaffected: %q", got)
	}
}

// Turning it on edits ACLs on the user's disk, so a cloned repository cannot.
func TestAProjectFileCannotTurnTheWindowsSandboxOnOrOff(t *testing.T) {
	cfg, _ := loadScoped(t, "", "[sandbox]\nwindows_integrity = true\n")
	if cfg.Sandbox.WindowsIntegrity {
		t.Fatal("a cloned repo's reasonix.toml switched the Windows sandbox on")
	}
	if !slices.Contains(ignoredKeys(cfg), "sandbox.windows_integrity") {
		t.Fatalf("ignored = %v, want sandbox.windows_integrity reported", ignoredKeys(cfg))
	}
	cfg, _ = loadScoped(t, "[sandbox]\nwindows_integrity = true\n", "[sandbox]\nwindows_integrity = false\n")
	if !cfg.Sandbox.WindowsIntegrity {
		t.Fatal("the project file switched off what the user turned on")
	}
}
