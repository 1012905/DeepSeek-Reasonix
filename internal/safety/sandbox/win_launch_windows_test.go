//go:build windows

package sandbox

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
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
	// A file that exists before the grant: the sandbox must be able to edit it.
	if err := os.WriteFile(filepath.Join(h.ws, "before.txt"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
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
	return h.runArgv([]string{"cmd", "/c", cmdline})
}

func (h *winHarness) runArgv(args []string) (string, int) {
	h.t.Helper()
	argv, wrapped := CommandArgs(h.spec, args)
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

// Inside a granted root the tools a developer uses still work under the token.
func TestRestrictedCommandsRunGitAndPowerShellInsideTheRoot(t *testing.T) {
	h := newWinHarness(t)
	if out, code := h.run(`cd /d ` + h.ws + ` && git init -q g && echo gitok`); code != 0 || !strings.Contains(out, "gitok") || !exists(filepath.Join(h.ws, "g", ".git")) {
		t.Errorf("git init: code %d out %q", code, out)
	}
	target := filepath.Join(h.ws, "ps.txt")
	if out, code := h.runArgv([]string{"powershell", "-NoProfile", "-Command", "Set-Content -Path '" + target + "' -Value x; 'psok'"}); code != 0 || !strings.Contains(out, "psok") || !exists(target) {
		t.Errorf("powershell write: code %d out %q", code, out)
	}
	keep := filepath.Join(h.out, "keep.txt")
	if err := os.WriteFile(keep, []byte("k"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.runArgv([]string{"powershell", "-NoProfile", "-Command", "Remove-Item -LiteralPath '" + keep + "' -Force"})
	if !exists(keep) {
		t.Error("PowerShell removed a file outside the root")
	}
}

// GrantSpec prepares an enforcing spec through the real book: roots granted,
// token proven, SID set; and a spec that does not enforce is left alone.
func TestGrantSpecPreparesAnEnforcingSpecAndLeavesAnOffSpecAlone(t *testing.T) {
	// The caches it grants are the user's real ones unless the profile is moved.
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	state := t.TempDir()
	ws := t.TempDir()
	off := GrantSpec(Spec{Mode: "off", WriteRoots: []string{ws}}, state)
	if off.GrantSID != "" || off.GrantFault != nil {
		t.Fatalf("an off spec was touched: %+v", off)
	}
	on := GrantSpec(Spec{Mode: "enforce", WriteRoots: []string{ws}}, state)
	if on.GrantFault != nil || on.GrantSID == "" {
		t.Fatalf("enforce spec = fault %v sid %q", on.GrantFault, on.GrantSID)
	}
	if ok, _ := WindowsGrantOps().Granted(ws, on.GrantSID); !ok {
		t.Fatal("the workspace root was not granted")
	}
	if !AvailableFor(on) || bwrapFaultFor(on) != nil {
		t.Fatal("a prepared spec reports itself unavailable")
	}
	t.Cleanup(func() {
		if b, err := windowsBook(state); err == nil {
			_ = b.Revoke(ws)
		}
	})
}

// A damaged record is a typed fault with its own stable code, never a fresh start.
func TestGrantSpecReportsADamagedRecordAsATypedFault(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	state := t.TempDir()
	if err := os.WriteFile(filepath.Join(state, "windows-grants.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec := GrantSpec(Spec{Mode: "enforce", WriteRoots: []string{t.TempDir()}}, state)
	if !errors.Is(spec.GrantFault, ErrWinGrantUnreadable) || spec.GrantSID != "" {
		t.Fatalf("spec = fault %v sid %q", spec.GrantFault, spec.GrantSID)
	}
	if AvailableFor(spec) {
		t.Fatal("an unprepared enforcing spec reports itself available")
	}
	if got := UnavailableCodeFor(spec); got != "sandbox.win_grant_unreadable" {
		t.Fatalf("code = %q", got)
	}
	if _, wrapped := CommandArgs(spec, []string{"cmd"}); wrapped {
		t.Fatal("a command was wrapped without a prepared spec")
	}
}

// labelLevel is the SID of a path's mandatory label, "" when it has none.
func labelLevel(t *testing.T, path string) string {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.LABEL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if ctrl, _, err := sd.Control(); err != nil || ctrl&windows.SE_SACL_PRESENT == 0 {
		return ""
	}
	sacl, _, err := sd.SACL()
	if err != nil || sacl == nil {
		return ""
	}
	for i := uint32(0); i < uint32(sacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(sacl, i, &ace) == nil && ace.Header.AceType == labelAceType {
			return (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		}
	}
	return ""
}

// Inside a root the sandbox edits and removes files that were already there, and
// what it creates is Medium like the user's own files: a program it builds runs
// at the user's level when the user starts it.
func TestRestrictedCommandsEditExistingFilesAndLeaveMediumFilesBehind(t *testing.T) {
	h := newWinHarness(t)
	before := filepath.Join(h.ws, "before.txt")
	h.run(`echo more>> ` + before)
	if b, _ := os.ReadFile(before); !strings.Contains(string(b), "more") {
		t.Errorf("the sandbox could not edit a file that existed before the grant: %q", b)
	}
	made := filepath.Join(h.ws, "made.exe")
	h.run(`echo MZ> ` + made)
	if lvl := labelLevel(t, made); lvl != mediumSID {
		t.Errorf("a file the sandbox made carries level %q, want Medium %s (a Low program would run Low for the user too)", lvl, mediumSID)
	}
	if lvl := labelLevel(t, before); lvl != mediumSID {
		t.Errorf("a file that existed before the grant carries level %q, want Medium", lvl)
	}
	h.run(`del ` + before)
	if exists(before) {
		t.Error("the sandbox could not remove a file inside its own root")
	}
}
