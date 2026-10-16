package shellsafe

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// For every generated command the classifier accepts as a read, run it in a
// scratch directory and compare the directory before and after: any change is
// a command called a read that wrote.

var differentialPrograms = []string{
	"cat", "head", "tail", "ls", "find", "grep", "egrep", "fgrep", "echo", "printf", "pwd",
	"wc", "sort", "uniq", "cut", "tr", "stat", "file", "du", "diff", "cmp", "comm",
	"basename", "dirname", "realpath", "readlink", "env", "true", "false", "test",
}

var differentialArgs = []string{
	"a.txt", "b.txt", "out.txt", "new.txt", "sub", "sub/c.txt", ".", "-", "--",
	"-n", "-l", "-r", "-i", "-v", "-c", "-d", "-u", "-f", "-o", "-s", "-t", "-e", "-x", "-a", "-z", "-1", "-2", "-3",
	"-delete", "-fprint", "-fprintf", "-fls", "-exec", "-execdir", "-ok", "-okdir", "-name", "-type", "-print", "-print0", "-prune",
	"--output=out.txt", "--output", "--compress-program=sh", "--files0-from=a.txt", "--help", "--version", "--color=always",
	"rm", "touch", "mv", "cp", "sh", "-c", "{}", ";", "+", "tee", "x", "1", "2", "pattern", "'a b'", `"a.txt"`,
}

func snapshotDir(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		line := fmt.Sprintf("%s|%v|%d", rel, info.Mode(), info.Size())
		if info.Mode().IsRegular() {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			line += fmt.Sprintf("|%x", sha256.Sum256(data))
		}
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func seedScratch(t *testing.T, dir string) {
	t.Helper()
	for name, body := range map[string]string{"a.txt": "alpha\nbeta\nalpha\n", "b.txt": "beta\ngamma\n", "sub/c.txt": "pattern here\n"} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEveryCommandTheClassifierCallsAReadLeavesTheDirectoryAlone(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a process per generated command")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the generated commands are POSIX shell commands")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	present := map[string]bool{}
	for _, program := range differentialPrograms {
		if _, err := exec.LookPath(program); err == nil || program == "echo" || program == "printf" || program == "test" || program == "true" || program == "false" {
			present[program] = true
		}
	}
	rng := rand.New(rand.NewSource(20261016))
	scratch := t.TempDir()
	home := t.TempDir()
	classified, executed := 0, 0
	seen := map[string]bool{}
	for range 1200 {
		program := differentialPrograms[rng.Intn(len(differentialPrograms))]
		if !present[program] {
			continue
		}
		parts := []string{program}
		for n := rng.Intn(5); n > 0; n-- {
			parts = append(parts, differentialArgs[rng.Intn(len(differentialArgs))])
		}
		command := strings.Join(parts, " ")
		if seen[command] {
			continue
		}
		seen[command] = true
		base, sub, fields, ok := ClassifyReadOnlyCommand(command)
		if !ok || ArgsMakeReadOnlyCommandWrite(base, sub, fields) {
			continue
		}
		classified++
		if err := os.RemoveAll(scratch); err != nil {
			t.Fatal(err)
		}
		seedScratch(t, scratch)
		before := snapshotDir(t, scratch)
		cmd := exec.Command(sh, "-c", command)
		cmd.Dir = scratch
		cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "LC_ALL=C"}
		cmd.Stdin = nil
		done := make(chan struct{})
		go func() {
			_ = cmd.Run()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		executed++
		if after := snapshotDir(t, scratch); after != before {
			t.Errorf("classified as a read but changed the directory: %q\nbefore:\n%s\nafter:\n%s", command, before, after)
		}
	}
	if classified < 30 {
		t.Fatalf("only %d generated commands were classified as reads; the generator no longer exercises the table", classified)
	}
	t.Logf("%d commands classified as reads, %d executed", classified, executed)
}
