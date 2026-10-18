//go:build windows

package sandbox

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type winHarness struct {
	t             *testing.T
	book          *GrantBook
	ws, out, temp string
	spec          Spec
}

// newWinHarness grants a workspace through the real book and returns a spec
// that runs commands under the real launcher. It needs the interactive desktop
// session: a network logon has no logon SID to restrict with.
func newWinHarness(t *testing.T) *winHarness {
	t.Helper()
	base := t.TempDir()
	h := &winHarness{t: t, ws: filepath.Join(base, "ws"), out: filepath.Join(base, "outside")}
	h.temp = filepath.Join(h.ws, "tmp")
	for _, d := range []string{h.ws, h.out, h.temp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	book, err := OpenGrantBook(filepath.Join(base, "state"), WindowsGrantOps())
	if err != nil {
		t.Fatal(err)
	}
	if err := book.Ensure(h.ws); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = book.Revoke(h.ws) })
	h.book = book
	h.spec = Spec{Mode: "enforce", WriteRoots: []string{h.ws}, GrantSID: book.SID(), SessionTemp: h.temp}
	return h
}

func (h *winHarness) run(cmdline string) (string, int) {
	h.t.Helper()
	argv, wrapped := CommandArgs(h.spec, []string{"cmd", "/c", cmdline})
	if !wrapped {
		h.t.Fatal("the command was not wrapped")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		h.t.Fatalf("launcher did not run: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), code
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

// What the host observes, not exit codes: a denied write leaves no file.
func TestRestrictedCommandsWriteOnlyWhereTheSIDWasGranted(t *testing.T) {
	h := newWinHarness(t)
	existing := filepath.Join(h.out, "existing.txt")
	if err := os.WriteFile(existing, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(h.ws, "jn")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, h.out).CombinedOutput(); err != nil {
		t.Skipf("cannot make a junction here: %v %s", err, out)
	}
	hkcu := `HKCU\Software\rxsandbox-probe`
	cmds := []string{
		`echo ok> ` + filepath.Join(h.ws, "inside.txt"),
		`echo no> ` + filepath.Join(h.out, "outside.txt"),
		`echo no>> ` + existing,
		`del ` + existing,
		`mkdir ` + filepath.Join(h.out, "newdir"),
		`echo no> ` + filepath.Join(junction, "via-junction.txt"),
		`reg add ` + hkcu + ` /v a /d b /f`,
		`cmd /c "echo nested> ` + filepath.Join(h.out, "nested.txt") + `"`,
	}
	t.Cleanup(func() { _ = exec.Command("reg", "delete", hkcu, "/f").Run() })
	for _, c := range cmds {
		out, code := h.run(c)
		if b, err := os.ReadFile(existing); err != nil || string(b) != "secret" {
			t.Errorf("after %q (exit %d, %q) the file outside the root is %q (%v)", c, code, out, b, err)
			_ = os.WriteFile(existing, []byte("secret"), 0o644)
		}
	}
	if !exists(filepath.Join(h.ws, "inside.txt")) {
		t.Error("a write inside the granted root did not land")
	}
	for _, p := range []string{"outside.txt", "newdir", "via-junction.txt", "nested.txt"} {
		if exists(filepath.Join(h.out, p)) {
			t.Errorf("a restricted command created %s outside the root", p)
		}
	}
	if b, err := os.ReadFile(existing); err != nil || string(b) != "secret" {
		t.Errorf("a restricted command changed or removed a file outside the root: %q %v", b, err)
	}
	if err := exec.Command("reg", "query", hkcu).Run(); err == nil {
		t.Error("a restricted command wrote to HKCU")
	}
}

// Reads and the network are not claimed, and must keep working.
func TestRestrictedCommandsStillReadOutsideAndReportTheirExitStatus(t *testing.T) {
	h := newWinHarness(t)
	if err := os.WriteFile(filepath.Join(h.out, "readable.txt"), []byte("hello-outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := h.run(`type ` + filepath.Join(h.out, "readable.txt")); code != 0 || !strings.Contains(out, "hello-outside") {
		t.Errorf("read outside: code %d out %q", code, out)
	}
	if _, code := h.run(`exit 7`); code != 7 {
		t.Errorf("exit status = %d, want 7", code)
	}
	if out, _ := h.run(`echo %TEMP%`); !strings.EqualFold(out, h.temp) {
		t.Errorf("TEMP = %q, want the session temp %q", out, h.temp)
	}
}

// The job ends the whole tree when the launcher goes, so a child cannot outlive
// a cancelled command.
func TestCancellingTheLauncherEndsTheTree(t *testing.T) {
	h := newWinHarness(t)
	late := filepath.Join(h.ws, "late.txt")
	argv, _ := CommandArgs(h.spec, []string{"cmd", "/c", `ping -n 7 127.0.0.1 >nul & echo late> ` + late})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_ = exec.CommandContext(ctx, argv[0], argv[1:]...).Run()
	time.Sleep(8 * time.Second)
	if exists(late) {
		t.Fatal("a child outlived its cancelled launcher and finished its work")
	}
}
