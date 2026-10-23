// Package hostwrite names the host directories a jailed command may write
// besides its own write roots — temporary directories and toolchain caches —
// so the jail that grants them and the checks that must not trust what lies
// there read one list.
package hostwrite

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dirs lists those directories for this host, unresolved. Windows jails no
// command, so nothing there is narrowed and the list is empty.
func Dirs() []string {
	return dirsFor(runtime.GOOS)
}

// Caches lists the toolchain caches a build writes through, on every platform.
// A check that must not trust what a cache holds reads this rather than Dirs:
// Dirs is what a jail grants, and a host that jails nothing grants nothing.
func Caches() []string {
	var dirs []string
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		dirs = append(dirs, dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return dirs
	}
	for _, sub := range []string{".cargo", ".npm", "go"} {
		dirs = append(dirs, filepath.Join(home, sub))
	}
	return dirs
}

func dirsFor(goos string) []string {
	home, homeErr := os.UserHomeDir()
	switch goos {
	case "windows":
		return nil
	case "darwin":
		dirs := []string{"/tmp", "/private/tmp", "/private/var/folders", os.TempDir()}
		if homeErr == nil {
			// go build/test → Library/Caches + go; pip/etc → .cache; npm/cargo too.
			for _, sub := range []string{"Library/Caches", ".cache", ".npm", ".cargo", "go"} {
				dirs = append(dirs, filepath.Join(home, sub))
			}
		}
		return dirs
	default:
		var dirs []string
		if td := os.TempDir(); td != "" && td != "/tmp" {
			dirs = append(dirs, td)
		}
		if homeErr == nil {
			for _, sub := range []string{".cache", ".cargo", ".npm", "go"} {
				dirs = append(dirs, filepath.Join(home, sub))
			}
		}
		return dirs
	}
}
