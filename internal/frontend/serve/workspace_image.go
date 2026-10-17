package serve

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxWorkspaceImage = 10 << 20

// imageTypes are what a document in the workspace may show inline. The type is
// decided by extension here, never sniffed, so a file cannot choose to be HTML.
var imageTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".svg":  "image/svg+xml",
}

// workspaceImage serves an image a rendered document references. The response
// is sandboxed: an SVG opened directly at this address is a document on the
// kernel's origin, and must not run the script it may carry.
func (s *Server) workspaceImage(w http.ResponseWriter, r *http.Request) {
	path, err := workspacePath(r.URL.Query().Get("path"))
	if err != nil {
		refuse(w, http.StatusBadRequest, "workspace.path_outside_tree", err.Error(), nil)
		return
	}
	contentType, ok := imageTypes[strings.ToLower(filepath.Ext(path))]
	if !ok {
		refuse(w, http.StatusBadRequest, "workspace.file_unreadable", "not an image this view shows", nil)
		return
	}
	base, ok := s.readRoot(w, r.URL.Query().Get("root"))
	if !ok {
		return
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		refuse(w, http.StatusInternalServerError, "workspace.file_failed", err.Error(), nil)
		return
	}
	defer root.Close()
	file, err := root.Open(path)
	if err != nil {
		refuseOpen(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() || info.Size() > maxWorkspaceImage {
		refuse(w, http.StatusBadRequest, "workspace.file_unreadable", "not a file, or larger than 10 MiB", nil)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = io.Copy(w, io.LimitReader(file, maxWorkspaceImage))
}

// readRoot is the folder a read comes from: the workspace, or a folder the user
// granted this workspace, named by its own path. A grant lives in the user's
// home, so a checkout cannot add one to itself; a folder that is not on that
// list, or no longer passes the vetting a grant had to pass, is refused.
func (s *Server) readRoot(w http.ResponseWriter, named string) (string, bool) {
	primary := s.ctl().WorkspaceRoot()
	if strings.TrimSpace(named) == "" || filepath.Clean(named) == filepath.Clean(primary) {
		return primary, true
	}
	extras, err := s.ctl().ReadableExtraRoots()
	if err != nil {
		refuse(w, http.StatusInternalServerError, "workspace.file_failed", err.Error(), nil)
		return "", false
	}
	for _, extra := range extras {
		if filepath.Clean(extra) == filepath.Clean(named) {
			return extra, true
		}
	}
	refuse(w, http.StatusForbidden, "workspace.root_not_granted", "that folder is not one this workspace was granted", map[string]any{"root": named})
	return "", false
}
