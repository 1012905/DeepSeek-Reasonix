package control

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/state/memory"
)

func externalEditSession(t *testing.T) (*Controller, string) {
	t.Helper()
	root := testenv.TempDir(t)
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	c := New(Options{Memory: memory.Load(memory.Options{CWD: cwd, UserDir: filepath.Join(root, "user")})})
	path, err := c.SaveMemory(memory.Memory{
		Name:        "deploy-target",
		Description: "where the staging deploy goes",
		Keywords:    "staging deploy target",
		Type:        memory.TypeProject,
		Body:        "old body marker",
	})
	if err != nil {
		t.Fatalf("SaveMemory: %v", err)
	}
	// The save queues a note that suppresses recall for one turn; spend it.
	c.Compose("hello")
	return c, path
}

func rewriteOnDisk(t *testing.T, path, from, to string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), from) {
		t.Fatalf("fact file has no %q:\n%s", from, raw)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A fact the last recall returned is checked against its file before the next
// recall, so an edit made outside this session (by the person, another process
// or another session) is what the model reads, not the snapshot loaded earlier.
func TestRecallReadsAFactEditedOutsideTheSession(t *testing.T) {
	c, path := externalEditSession(t)
	if first := c.Compose("staging deploy target"); !strings.Contains(first, "old body marker") {
		t.Fatalf("the probe proves nothing unless the turn carries the fact:\n%s", first)
	}
	rewriteOnDisk(t, path, "old body marker", "new body marker")
	next := c.Compose("staging deploy target")
	if !strings.Contains(next, "new body marker") || strings.Contains(next, "old body marker") {
		t.Fatalf("the turn still carries the stale body:\n%s", next)
	}
}

func TestRecallStopsOfferingAFactDeletedOutsideTheSession(t *testing.T) {
	c, path := externalEditSession(t)
	if first := c.Compose("staging deploy target"); !strings.Contains(first, "old body marker") {
		t.Fatalf("setup: %s", first)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if next := c.Compose("staging deploy target"); strings.Contains(next, "old body marker") {
		t.Fatalf("a deleted fact is still recalled:\n%s", next)
	}
}

// Re-checking is bounded to what the last recall returned: a fact edited into
// relevance that was never a hit stays invisible until the session reloads, and
// an untouched hit costs no reload.
func TestAnUntouchedRecalledFactDoesNotReloadTheSnapshot(t *testing.T) {
	c, _ := externalEditSession(t)
	if first := c.Compose("staging deploy target"); !strings.Contains(first, "old body marker") {
		t.Fatalf("setup: %s", first)
	}
	before := c.Memory()
	c.Compose("staging deploy target")
	if c.Memory() != before {
		t.Fatal("an unchanged recalled fact reloaded the whole snapshot")
	}
}
