package sessionstore

import (
	"errors"
	"os"
	"path/filepath"
)

// copySessionFile copies one message-format session into the import, reporting
// whether it landed. A copy that fails has to reach the caller: the one-time
// import marker is only truthful when every session arrived.
func copySessionFile(jsonlPath, dest string) bool {
	if isNativeSessionEventLog(SessionEventLogPath(jsonlPath)) {
		return saveNativeSessionCopy(jsonlPath, dest) == nil
	}
	return transformAndCopyJsonl(jsonlPath, dest) == nil
}

// publishFileNoReplace atomically publishes a completed sibling temp file
// without replacing a destination another startup/import writer created.
// The temp and destination share a directory, so a hard link is atomic and
// portable across the filesystems Reasonix supports.
func publishFileNoReplace(tmp, dst string) error {
	if err := linkFileNoReplace(tmp, dst); err != nil {
		return err
	}
	return os.Remove(tmp)
}

func linkFileNoReplace(src, dst string) error {
	if err := os.Link(src, dst); err != nil {
		if os.IsExist(err) {
			return os.ErrExist
		}
		return err
	}
	return nil
}

// copyBranchMetaSidecar copies <src>.meta to <dst>.meta when the source has one,
// so the sidebar keeps the session's scope, archive mark, pin, title, and tree
// position. A source without a sidecar has nothing to carry — that session keeps
// showing with a generated title — but a failed write is returned so the caller
// can leave the import retryable instead of dropping the meaning silently.
func copyBranchMetaSidecar(srcPath, dstPath string) error {
	b, err := os.ReadFile(BranchMetaPath(srcPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	dstMeta := BranchMetaPath(dstPath)
	if err := os.MkdirAll(filepath.Dir(dstMeta), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dstMeta), ".branch.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, dstMeta); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// completeBranchMetaSidecar finishes a transcript whose sidecar an interrupted
// import never published. It reports whether it wrote one. A sidecar file that is
// already there is complete and is never rewritten, so the ordinary retry of a
// finished session copies nothing.
func completeBranchMetaSidecar(srcPath, dstPath string) (bool, error) {
	if info, err := os.Stat(BranchMetaPath(dstPath)); err == nil && !info.IsDir() {
		return false, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := copyBranchMetaSidecar(srcPath, dstPath); err != nil {
		return false, err
	}
	return true, nil
}
