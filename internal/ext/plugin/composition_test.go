package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/safety/sandbox"
)

func TestCompositionProcessStateIsPrivateAndConfinedToOwnedSubdirectory(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"one", "two"} {
		root := filepath.Join(base, "sessions", name)
		_, env, err := prepareMCPPrivateStateForOS(Spec{StateDir: base, ProcessStateDir: root}, sandbox.Spec{}, nil, "linux")
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := envValue(env, "XDG_STATE_HOME"); got != filepath.Join(root, "state") {
			t.Fatalf("state=%s", got)
		}
	}
	for _, path := range []string{base, filepath.Join(base, "sessions"), filepath.Join(base, "sessions", "..", "outside"), filepath.Join(base, "sessions", "nested", "child")} {
		if _, _, err := prepareMCPPrivateStateForOS(Spec{StateDir: base, ProcessStateDir: path}, sandbox.Spec{}, nil, "linux"); err == nil {
			t.Fatalf("unsafe state accepted: %s", path)
		}
	}
	keep := filepath.Join(base, "credential-reference")
	if err := os.WriteFile(keep, []byte("shared"), 0o600); err != nil {
		t.Fatal(err)
	}
	transport := &stdioTransport{privateState: filepath.Join(base, "sessions", "one")}
	transport.close()
	if _, err := os.Stat(transport.privateState); !os.IsNotExist(err) {
		t.Fatalf("private state remains: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("credential store removed")
	}
}

func TestCompositionProcessStateRejectsSymlinkOwner(t *testing.T) {
	base, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(base, "sessions")); err != nil {
		t.Fatal(err)
	}
	if _, err := CompositionProcessRoot(Spec{StateDir: base, ProcessStateDir: filepath.Join(base, "sessions", "session")}); err == nil {
		t.Fatal("symlink process state accepted")
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("outside directory changed: %v %v", entries, err)
	}
}
