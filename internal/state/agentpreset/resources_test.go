package agentpreset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRetainedDirectorySurvivesSourceChanges(t *testing.T) {
	s := New(t.TempDir(), t.TempDir())
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "playbook.md"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	asset, err := CaptureDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "playbook.md"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := s.Materialize(asset)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "playbook.md"))
	if err != nil || string(b) != "original" {
		t.Fatalf("asset %s %v", b, err)
	}
}

func TestAssetCannotEscapeItsManagedDirectory(t *testing.T) {
	s := New(t.TempDir(), t.TempDir())
	for _, file := range []AssetFile{{Path: "../outside", Data: []byte("bad")}, {Path: "link", Link: "../../outside"}} {
		if _, err := s.Materialize(DirectorySnapshot{Files: []AssetFile{file}}); err == nil {
			t.Fatalf("escaped asset: %+v", file)
		}
	}
}
