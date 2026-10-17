package control

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SaveAttachmentStreamInRoot(ctx context.Context, root, origName string, src io.Reader) (string, error) {
	ext := strings.ToLower(filepath.Ext(origName))
	if !safeAttachmentExt.MatchString(ext) {
		ext = ".bin"
	}
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", attachmentWriteError(err)
	}
	if err = ensureAttachmentRootIn(abs); err != nil {
		return "", attachmentWriteError(err)
	}
	rel, f, err := createAttachmentFileIn(abs, ext)
	if err != nil {
		return "", attachmentWriteError(err)
	}
	complete := false
	defer func() {
		_ = f.Close()
		if !complete {
			_ = os.Remove(filepath.Join(abs, rel))
		}
	}()
	n, err := io.CopyBuffer(f, attachmentContextReader{ctx, src}, make([]byte, 64<<10))
	if err != nil {
		return "", attachmentWriteError(err)
	}
	if err = ctx.Err(); err != nil {
		return "", attachmentWriteError(err)
	}
	if n == 0 {
		return "", attachmentSizeError(0, maxFileAttachmentBytes)
	}
	if err = f.Close(); err != nil {
		return "", attachmentWriteError(err)
	}
	complete = true
	return filepath.ToSlash(rel), nil
}

type attachmentContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r attachmentContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
