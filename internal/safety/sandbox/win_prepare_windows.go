//go:build windows

package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	windowsBooksMu sync.Mutex
	windowsBooks   = map[string]*GrantBook{}
)

// windowsBook opens the grant book under stateDir once per process and repairs
// it on the way in: what a crash left half-done is finished before a command runs.
func windowsBook(stateDir string) (*GrantBook, error) {
	windowsBooksMu.Lock()
	defer windowsBooksMu.Unlock()
	if b, ok := windowsBooks[stateDir]; ok {
		return b, nil
	}
	b, err := OpenGrantBook(stateDir, WindowsGrantOps())
	if err != nil {
		return nil, err
	}
	b.Repair()
	windowsBooks[stateDir] = b
	return b, nil
}

// GrantSpec prepares an enforcing spec for the Windows launcher: it grants the
// install's SID on the roots the spec names and the toolchain caches that exist
// (once each; later calls find the grant in place), and proves a restricted
// token can be made here. On any failure the spec comes back unprepared, with the
// reason as an ErrWin* identity, so a caller in enforce mode refuses instead of
// running unconfined. A spec that does not enforce is returned as it was.
func GrantSpec(spec Spec, stateDir string) Spec {
	if spec.Mode != "enforce" {
		return spec
	}
	book, err := windowsBook(stateDir)
	if err != nil {
		spec.GrantFault = fmt.Errorf("%w: %w", ErrWinGrantUnreadable, err)
		return spec
	}
	for _, dir := range planWriteRoots(spec, backendWriteDirs(spec)).dirs {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		if err := book.Ensure(dir); err != nil {
			spec.GrantFault = fmt.Errorf("%w: %s: %w", ErrWinGrantDenied, dir, err)
			return spec
		}
	}
	if err := probeToken(book.SID()); err != nil {
		spec.GrantFault = err
		return spec
	}
	spec.GrantSID = book.SID()
	return spec
}

// probeToken makes and drops a restricted token, to learn at session start what
// a launch would otherwise learn when it fails.
func probeToken(sid string) error {
	tok, err := restrictedToken(sid)
	if err != nil {
		if errors.Is(err, errNoLogonSID) {
			return fmt.Errorf("%w: %w", ErrWinNoLogonSID, err)
		}
		return fmt.Errorf("%w: %w", ErrWinLauncher, err)
	}
	tok.Close()
	return nil
}

// windowsCacheDirs are the per-user tool caches a build writes: the same job the
// Unix toolchain-cache list does. Each is granted only if it already exists.
func windowsCacheDirs() []string {
	var dirs []string
	local, home := os.Getenv("LOCALAPPDATA"), os.Getenv("USERPROFILE")
	for _, p := range []struct{ base, sub string }{
		{local, "go-build"}, {local, "npm-cache"}, {local, "pip"}, {local, "NuGet"},
		{home, "go"}, {home, ".cargo"}, {home, ".nuget"}, {home, ".cache"},
	} {
		if p.base != "" {
			dirs = append(dirs, filepath.Join(p.base, p.sub))
		}
	}
	return dirs
}
