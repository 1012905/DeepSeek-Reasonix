package memory

import (
	"fmt"
	"os"
)

// DirectoryReadError identifies an incomplete store read and preserves its OS cause.
type DirectoryReadError struct {
	Path string
	Err  error
}

func (e *DirectoryReadError) Error() string {
	return fmt.Sprintf("read memory directory %q: %v", e.Path, e.Err)
}

func (e *DirectoryReadError) Unwrap() error { return e.Err }

func readMemoryDirectory(path string) ([]os.DirEntry, error) {
	if path == "" {
		return nil, nil
	}
	err := fault("read-dir", path)
	var entries []os.DirEntry
	if err == nil {
		entries, err = os.ReadDir(path)
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, &DirectoryReadError{Path: path, Err: err}
	}
	return entries, nil
}
