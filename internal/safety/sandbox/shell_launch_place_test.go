package sandbox

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestShellLaunchPlace(t *testing.T) {
	for p, want := range map[string]bool{
		`C:\Git`:          true,
		`d:/Tools`:        true,
		`\\srv\share\Git`: false,
		`//srv/share/Git`: false,
		`\\?\UNC\srv\x`:   false,
		`\\.\pipe\x`:      false,
		`\??\UNC\srv\x`:   false,
		`Git`:             false,
		`C:Git`:           false,
		``:                false,
	} {
		if got := shellLaunchPlace(p); got != want {
			t.Errorf("shellLaunchPlace(%q) = %v, want %v", p, got, want)
		}
	}
}

// One hostile root, fed to every source of bash candidates and to the pinned
// path, never reaches a candidate list or a launch.
func TestEverySourceRefusesANetworkShareRoot(t *testing.T) {
	const share = `\\srv\share`
	env := map[string]string{"ProgramFiles": share, "ProgramW6432": share, "LOCALAPPDATA": share, "PATH": share}
	look := func(string) (string, error) { return share + `\Git\cmd\git.exe`, nil }
	for _, c := range bashCandidates(func(k string) string { return env[k] }, look) {
		if strings.Contains(c, "srv") {
			t.Errorf("a network share became a bash candidate: %q", c)
		}
	}
	if got := appendGitInstallRoots(nil, []string{share + `\Git`}); len(got) != 0 {
		t.Errorf("the registry source took a network root: %q", got)
	}
	pinned := share + `\bash.exe`
	launched := false
	if err := verifyShell("bash", pinned, func(string) bool { return true }, func(string) bool { launched = true; return true }, nil); err == nil || launched {
		t.Errorf("VerifyShell accepted a pinned network path (err=%v, launched=%v)", err, launched)
	}
	got := resolveShell("bash", pinned, nil, "windows", func(string) (string, error) { return "", filepath.ErrBadPattern },
		func(string) bool { return true }, nil, nil, func(string) bool { launched = true; return true }, func(string) bool { return false }, nil)
	if launched || got.Path == pinned {
		t.Errorf("a pinned network path was launched or chosen: %+v", got)
	}
}
