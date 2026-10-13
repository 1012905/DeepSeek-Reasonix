package serve

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"

	"reasonix/internal/contract/event"
	"reasonix/internal/contract/eventwire"
	"reasonix/internal/state/store"
)

// Persist identity only: plugin values and form defaults never enter the log.
func surfaceMarker(raw []byte) ([]byte, bool) {
	var w eventwire.Event
	if json.Unmarshal(raw, &w) != nil || w.Extension == nil || w.Extension.Kind == event.ExtensionSurfaceNotification {
		return nil, false
	}
	p := w.Extension
	w = eventwire.Event{Kind: w.Kind, Extension: &eventwire.ExtensionSurface{Code: event.SurfaceExpired, PluginID: p.PluginID, SurfaceID: p.SurfaceID, SessionID: p.SessionID, Generation: p.Generation, Kind: p.Kind}}
	raw, err := json.Marshal(w)
	return raw, err == nil
}

func (s *Server) extensionSurfaces(w http.ResponseWriter, r *http.Request) {
	s.bindMu.Lock()
	defer s.bindMu.Unlock()
	ctl := s.ctl()
	path := ctl.SessionPath()
	found := map[string]*eventwire.ExtensionSurface{}
	if path != "" {
		s.wire.mu.Lock()
		data, err := os.ReadFile(store.SessionWireLog(path))
		truncated, witnessErr := truncationWitnessed(store.SessionWireLogMeta(path))
		s.wire.mu.Unlock()
		if err != nil && !errors.Is(err, fs.ErrNotExist) || witnessErr != nil || truncated {
			refuse(w, http.StatusConflict, "surface.replay_incomplete", "extension surface history is unavailable or incomplete", nil)
			return
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev eventwire.Event
			if json.Unmarshal([]byte(line), &ev) != nil {
				refuse(w, http.StatusConflict, "surface.replay_incomplete", "extension surface history is unreadable", nil)
				return
			}
			if ev.Extension == nil || ev.Extension.Kind == event.ExtensionSurfaceNotification {
				continue
			}
			p := ev.Extension
			found[p.PluginID+"\x00"+p.SurfaceID] = &eventwire.ExtensionSurface{Code: event.SurfaceExpired, PluginID: p.PluginID, SurfaceID: p.SurfaceID, SessionID: p.SessionID, Generation: p.Generation, Kind: p.Kind}
		}
	}
	for _, p := range ctl.ExtensionSurfaces() {
		found[p.PluginID+"\x00"+p.SurfaceID] = eventwire.ToWireExtensionSurface(&p)
	}
	keys := make([]string, 0, len(found))
	for key := range found {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]*eventwire.ExtensionSurface, 0, len(keys))
	for _, key := range keys {
		out = append(out, found[key])
	}
	writeJSON(w, out)
}
