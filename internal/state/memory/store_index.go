package memory

import (
	"os"
	"path/filepath"
	"strings"

	fileencoding "reasonix/internal/base/fileutil/encoding"
)

// memoryFault, when non-nil, may fail one durable step. It is the single test
// seam for this package: production leaves it nil, and tests set it through
// withFault, which restores the previous value.
var memoryFault func(op, path string) error

func fault(op, path string) error {
	if memoryFault == nil {
		return nil
	}
	return memoryFault(op, path)
}

// readIndexIn treats a missing index as empty, but does not hide other read errors.
func readIndexIn(dir string) ([]byte, error) {
	if err := fault("read-index", filepath.Join(dir, indexFile)); err != nil {
		return nil, err
	}
	existing, err := fileencoding.ReadFileUTF8(filepath.Join(dir, indexFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return existing, err
}

// indexLinesExceptIn returns the managed MEMORY.md lines keyed by filename stem,
// dropping the entry for name and reporting whether it was present.
func indexLinesExceptIn(dir, name string) (map[string]string, bool, error) {
	existing, err := readIndexIn(dir)
	if err != nil {
		return nil, false, err
	}
	keep := map[string]string{}
	contains := false
	for line := range strings.SplitSeq(string(existing), "\n") {
		if mt := indexLineRe.FindStringSubmatch(line); mt != nil {
			if mt[1] == name {
				contains = true
			} else {
				keep[mt[1]] = strings.TrimRight(line, "\r")
			}
		}
	}
	return keep, contains, nil
}
