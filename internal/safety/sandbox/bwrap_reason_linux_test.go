//go:build linux

package sandbox

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBwrap puts a bwrap on PATH that exits with code, and nothing else.
func fakeBwrap(t *testing.T, code string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\nexit " + code + "\n"
	if err := os.WriteFile(filepath.Join(dir, "bwrap"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// Bubblewrap that is installed but cannot enter a namespace (a default
// container, a hardened kernel) is a different state from no bubblewrap at
// all, and the way out is different too.
func TestAnInstalledBwrapThatCannotEnterANamespaceIsNotReportedMissing(t *testing.T) {
	fakeBwrap(t, "1")
	if Available() {
		t.Fatal("a bwrap whose probe fails was reported available")
	}
	if err := UnavailableReason(); !errors.Is(err, ErrBwrapNamespaceDenied) || errors.Is(err, ErrBwrapMissing) {
		t.Fatalf("reason = %v, want the namespace-denied identity", err)
	}
	if got := UnavailableCode(); got != "sandbox.bwrap_namespace_denied" {
		t.Fatalf("code = %q", got)
	}
	hint := UnavailableRemediation()
	if strings.Contains(hint, "Install bubblewrap") {
		t.Fatalf("the hint tells the reader to install what is installed: %q", hint)
	}
	for _, want := range []string{"cannot create a namespace", "CAP_SYS_ADMIN", `[sandbox] bash = "off"`, "only boundary"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("hint = %q, want it to say %q", hint, want)
		}
	}
}

func TestNoBwrapOnPathIsReportedMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if Available() {
		t.Fatal("an absent bwrap was reported available")
	}
	if err := UnavailableReason(); !errors.Is(err, ErrBwrapMissing) || errors.Is(err, ErrBwrapNamespaceDenied) {
		t.Fatalf("reason = %v, want the missing identity", err)
	}
	if got := UnavailableCode(); got != "sandbox.no_bubblewrap" {
		t.Fatalf("code = %q", got)
	}
	if !strings.Contains(UnavailableRemediation(), "Install bubblewrap") {
		t.Fatalf("hint = %q, want the install step", UnavailableRemediation())
	}
}

func TestAUsableBwrapHasNoUnavailableReason(t *testing.T) {
	fakeBwrap(t, "0")
	if !Available() {
		t.Fatal("a bwrap whose probe succeeds was reported unavailable")
	}
	if err := UnavailableReason(); err != nil {
		t.Fatalf("reason = %v, want none", err)
	}
}

// Naming the cause never relaxes anything: with the namespace denied, an
// enforced command is still handed back unwrapped for the caller to refuse.
func TestADeniedNamespaceStillLeavesTheCommandUnwrapped(t *testing.T) {
	fakeBwrap(t, "1")
	argv, wrapped := Command(Spec{Mode: "enforce"}, Shell{}, "echo hi")
	if wrapped {
		t.Fatalf("a command was wrapped by an unusable bwrap: %v", argv)
	}
}
