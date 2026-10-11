//go:build windows

package agent

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// A junction is neither a symlink nor a regular directory to Go, and the store
// reached through one is not the host's.
func TestJunctionOnTheWayToTheStoreStaysListed(t *testing.T) {
	f := newReceiptFixture(t)
	real := f.write(t, filepath.Join("src", "api.py"))
	link := filepath.Join(f.root, ".reasonix", "tasks", "x")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Dir(real)).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	written := filepath.Join(link, "api.py")
	f.record(written)
	requireListed(t, f, written)
}
