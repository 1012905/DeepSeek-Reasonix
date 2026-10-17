package control

import (
	"context"
	"io"
	"path/filepath"
	"strings"
)

func SaveAttachmentStreamInRoot(ctx context.Context, root, origName string, src io.Reader) (string, error) {
	ext := strings.ToLower(filepath.Ext(origName))
	if !safeAttachmentExt.MatchString(ext) {
		ext = ".bin"
	}
	owned, err := createOwnedAttachmentFile(root, ext)
	if err != nil {
		return "", attachmentWriteError(err)
	}
	defer owned.cleanup()

	n, err := io.CopyBuffer(owned, attachmentContextReader{ctx, src}, make([]byte, 64<<10))
	if err != nil {
		return "", attachmentWriteError(err)
	}
	if err = ctx.Err(); err != nil {
		return "", attachmentWriteError(err)
	}
	if n == 0 {
		return "", attachmentSizeError(0, maxFileAttachmentBytes)
	}
	if err = owned.finish(); err != nil {
		return "", attachmentWriteError(err)
	}
	return filepath.ToSlash(owned.rel), nil
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
