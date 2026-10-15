package gitcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A folder git will not open is told apart from one git cannot be asked about.
func TestAbsenceNamesWhyAMarkedFolderHasNoRepository(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ./nowhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Open(context.Background(), dir)
	if !errors.Is(err, ErrNotRepository) {
		t.Fatalf("Open of a folder git refuses = %v, want ErrNotRepository kept for callers that only ask that", err)
	}
	absent, detail := AbsenceOf(err)
	if absent != AbsenceRefused || detail == "" {
		t.Fatalf("absence = %q %q, want refused with what git printed", absent, detail)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	_, err = Open(ctx, dir)
	if absent, _ := AbsenceOf(err); absent != AbsenceUnavailable {
		t.Fatalf("a git that was not asked in time is %q (%v), want unavailable", absent, err)
	}
	if absent, _ := AbsenceOf(nil); absent != "" {
		t.Fatalf("no error is %q, want no absence", absent)
	}
}

func TestFirstLineIsOneBoundedPrintableLine(t *testing.T) {
	if got := firstLine([]byte("fatal: x\x1b[31m\nsecond line")); got != "fatal: x[31m" {
		t.Fatalf("firstLine = %q", got)
	}
	long := make([]byte, 500)
	for i := range long {
		long[i] = 'a'
	}
	if got := firstLine(long); len(got) != 160 {
		t.Fatalf("len = %d, want 160", len(got))
	}
}
