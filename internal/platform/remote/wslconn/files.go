package wslconn

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"reasonix/internal/platform/remote/sftpfs"
)

type files struct{ c *Conn }

var _ sftpfs.Files = (*files)(nil)

const (
	statScript = `[ -e "$1" ] || exit 44
exec stat -L -c '%s %f %Y' -- "$1"`

	readScript = `[ -e "$1" ] || exit 44
[ -d "$1" ] && exit 45
exec head -c "$2" -- "$1"`

	writeScript = `t="$(dirname -- "$1")/.$(basename -- "$1").reasonix-tmp-$$"
{ cat > "$t" && chmod "$2" "$t" && mv -f -- "$t" "$1"; } || { rm -f -- "$t"; exit 1; }`

	mkdirAllScript = `exec mkdir -p -- "$1"`
	mkdirScript    = `exec mkdir -- "$1"`
	chmodScript    = `[ -e "$1" ] || exit 44
exec chmod "$2" -- "$1"`

	removeScript = `[ -e "$1" ] || [ -L "$1" ] || exit 44
if [ -d "$1" ] && [ ! -L "$1" ]; then
  if [ "$2" = 1 ]; then exec rm -rf -- "$1"; else exec rmdir -- "$1"; fi
fi
exec rm -f -- "$1"`

	realPathScript = `case "$1" in
"~") p=$HOME ;;
"~/"*) p=$HOME/${1#"~/"} ;;
*) p=$1 ;;
esac
exec readlink -f -- "$p"`
)

func (f *files) Stat(ctx context.Context, p string) (sftpfs.Entry, error) {
	res, err := f.c.script(ctx, nil, statScript, p)
	if err != nil {
		return sftpfs.Entry{}, err
	}
	if err := failure("stat "+p, res); err != nil {
		return sftpfs.Entry{}, err
	}
	var size, mtime int64
	var rawMode uint64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(res.Stdout)), "%d %x %d", &size, &rawMode, &mtime); err != nil {
		return sftpfs.Entry{}, fmt.Errorf("stat %s: unreadable answer: %w", p, err)
	}
	mode := fileMode(uint32(rawMode))
	return sftpfs.Entry{Name: path.Base(p), Path: p, Size: size, Mode: mode, ModTime: mtime, IsDir: mode.IsDir()}, nil
}

// fileMode decodes a raw st_mode by its bit fields: the format bits say what
// the file is, the low twelve are the permissions.
func fileMode(raw uint32) fs.FileMode {
	mode := fs.FileMode(raw & 0o777)
	if raw&0o4000 != 0 {
		mode |= fs.ModeSetuid
	}
	if raw&0o2000 != 0 {
		mode |= fs.ModeSetgid
	}
	if raw&0o1000 != 0 {
		mode |= fs.ModeSticky
	}
	switch raw & 0o170000 {
	case 0o040000:
		mode |= fs.ModeDir
	case 0o120000:
		mode |= fs.ModeSymlink
	case 0o010000:
		mode |= fs.ModeNamedPipe
	case 0o140000:
		mode |= fs.ModeSocket
	case 0o020000:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case 0o060000:
		mode |= fs.ModeDevice
	}
	return mode
}

func (f *files) ReadFile(ctx context.Context, p string, maxSize int64) ([]byte, bool, sftpfs.Kind, error) {
	if maxSize <= 0 {
		maxSize = sftpfs.DefaultReadCap
	}
	res, err := f.c.script(ctx, nil, readScript, p, strconv.FormatInt(maxSize+1, 10))
	if err != nil {
		return nil, false, sftpfs.KindBinary, err
	}
	if err := failure("read "+p, res); err != nil {
		return nil, false, sftpfs.KindBinary, err
	}
	data, truncated := res.Stdout, false
	if int64(len(data)) > maxSize {
		data, truncated = data[:maxSize], true
	}
	return data, truncated, sftpfs.DetectKind(data), nil
}

func (f *files) WriteFileAtomic(ctx context.Context, p string, data []byte, perm fs.FileMode) error {
	res, err := f.c.script(ctx, bytes.NewReader(data), writeScript, p, strconv.FormatUint(uint64(perm.Perm()), 8))
	if err != nil {
		return err
	}
	return failure("write "+p, res)
}

func (f *files) MkdirAll(ctx context.Context, p string) error {
	return f.simple(ctx, "mkdir "+p, mkdirAllScript, p)
}

func (f *files) MkdirExclusive(ctx context.Context, p string) error {
	return f.simple(ctx, "mkdir "+p, mkdirScript, p)
}

func (f *files) Chmod(ctx context.Context, p string, perm fs.FileMode) error {
	return f.simple(ctx, "chmod "+p, chmodScript, p, strconv.FormatUint(uint64(perm.Perm()), 8))
}

func (f *files) Remove(ctx context.Context, p string, recursive bool) error {
	flag := "0"
	if recursive {
		flag = "1"
	}
	return f.simple(ctx, "remove "+p, removeScript, p, flag)
}

func (f *files) RealPath(ctx context.Context, p string) (string, error) {
	res, err := f.c.script(ctx, nil, realPathScript, p)
	if err != nil {
		return "", err
	}
	if err := failure("realpath "+p, res); err != nil {
		return "", err
	}
	return strings.TrimRight(string(res.Stdout), "\n"), nil
}

func (f *files) simple(ctx context.Context, op, body string, args ...string) error {
	res, err := f.c.script(ctx, nil, body, args...)
	if err != nil {
		return err
	}
	return failure(op, res)
}

// dirsScript prints the directories under $1 (symlinks to directories
// included), each name closed by a NUL so no file name can end it early.
const dirsScript = `cd -- "$1" 2>/dev/null || exit 44
for f in ./* ./.[!.]* ./..?*; do
  [ -d "$f" ] && printf '%s\0' "${f#./}"
done
exit 0`

// Dirs resolves dir (empty or ~ meaning the home directory) and lists the
// directories inside it.
func (c *Conn) Dirs(ctx context.Context, dir string) (string, []sftpfs.Entry, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "~"
	}
	fsys := &files{c: c}
	at, err := fsys.RealPath(ctx, dir)
	if err != nil {
		return "", nil, err
	}
	res, err := c.script(ctx, nil, dirsScript, at)
	if err != nil {
		return "", nil, err
	}
	if err := failure("list "+at, res); err != nil {
		return "", nil, err
	}
	var out []sftpfs.Entry
	for name := range strings.SplitSeq(string(res.Stdout), "\x00") {
		if name == "" {
			continue
		}
		out = append(out, sftpfs.Entry{Name: name, Path: path.Join(at, name), IsDir: true, Mode: fs.ModeDir})
	}
	return at, out, nil
}
