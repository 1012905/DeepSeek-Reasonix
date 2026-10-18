//go:build linux || windows

package wslconn

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"
	"testing"

	"reasonix/internal/platform/remote/sftpfs"
)

// runFileScenario drives the file layer against a real POSIX shell: the local
// one in CI, a distribution in the live test. dir is a scratch directory there.
func runFileScenario(t *testing.T, c *Conn, dir string) {
	t.Helper()
	ctx := context.Background()
	fsys, _ := c.Files()
	awkward := path.Join(dir, `a b "q" 'x' $HOME `+"`z`"+` ünï`)

	if err := fsys.MkdirAll(ctx, awkward+"/deep/er"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := fsys.MkdirExclusive(ctx, awkward+"/deep"); err == nil {
		t.Fatal("MkdirExclusive on an existing directory succeeded")
	}
	if err := fsys.MkdirExclusive(ctx, awkward+"/lock"); err != nil {
		t.Fatalf("MkdirExclusive: %v", err)
	}

	file := awkward + "/state.json"
	want := []byte("{\"port\": 4242}\n")
	if err := fsys.WriteFileAtomic(ctx, file, want, 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if err := fsys.WriteFileAtomic(ctx, file, append(want, 'x'), 0o600); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, truncated, kind, err := fsys.ReadFile(ctx, file, 0)
	if err != nil || truncated || kind != sftpfs.KindText || !bytes.Equal(got, append(want, 'x')) {
		t.Fatalf("ReadFile = %q, %v, %v, %v", got, truncated, kind, err)
	}
	if head, truncated, _, err := fsys.ReadFile(ctx, file, 4); err != nil || !truncated || string(head) != "{\"po" {
		t.Fatalf("a capped read = %q, %v, %v", head, truncated, err)
	}
	en, err := fsys.Stat(ctx, file)
	if err != nil || en.IsDir || en.Size != int64(len(want)+1) || en.Mode.Perm() != 0o600 || en.ModTime == 0 {
		t.Fatalf("Stat file = %+v, %v", en, err)
	}
	if en, err := fsys.Stat(ctx, awkward); err != nil || !en.IsDir {
		t.Fatalf("Stat dir = %+v, %v", en, err)
	}
	if _, _, _, err := fsys.ReadFile(ctx, awkward+"/deep", 0); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("reading a directory = %v, want a failure that is not 'missing'", err)
	}

	if err := fsys.Chmod(ctx, file, 0o640); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if en, _ := fsys.Stat(ctx, file); en.Mode.Perm() != 0o640 {
		t.Fatalf("mode after chmod = %v", en.Mode)
	}

	for name, call := range map[string]func() error{
		"stat":   func() error { _, err := fsys.Stat(ctx, awkward+"/none"); return err },
		"read":   func() error { _, _, _, err := fsys.ReadFile(ctx, awkward+"/none", 0); return err },
		"chmod":  func() error { return fsys.Chmod(ctx, awkward+"/none", 0o600) },
		"remove": func() error { return fsys.Remove(ctx, awkward+"/none", false) },
	} {
		if err := call(); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s of a missing path = %v, want fs.ErrNotExist", name, err)
		}
	}

	real, err := fsys.RealPath(ctx, awkward+"/deep/../state.json")
	if err != nil || !strings.HasSuffix(real, "/state.json") || strings.Contains(real, "..") {
		t.Fatalf("RealPath = %q, %v", real, err)
	}
	if home, err := fsys.RealPath(ctx, "~"); err != nil || !strings.HasPrefix(home, "/") {
		t.Fatalf("RealPath(~) = %q, %v", home, err)
	}

	if err := fsys.Remove(ctx, awkward+"/deep", false); err == nil {
		t.Fatal("a non-recursive remove of a non-empty directory succeeded")
	}
	if err := fsys.Remove(ctx, file, false); err != nil {
		t.Fatalf("Remove file: %v", err)
	}
	if err := fsys.Remove(ctx, awkward, true); err != nil {
		t.Fatalf("Remove tree: %v", err)
	}
	if _, err := fsys.Stat(ctx, awkward); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the removed tree is still there: %v", err)
	}

	res, err := c.Exec(ctx, "printf '%s' \"$((6*7))\"; echo err >&2; exit 3")
	if err != nil || string(res.Stdout) != "42" || strings.TrimSpace(string(res.Stderr)) != "err" || res.ExitCode != 3 {
		t.Fatalf("Exec = %+v, %v", res, err)
	}
}
