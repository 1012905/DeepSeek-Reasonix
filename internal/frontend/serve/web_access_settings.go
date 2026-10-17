package serve

import (
	"encoding/json"
	"errors"
	"net/http"

	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
)

func (s *Server) registerWebAccessRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /web-access-settings", s.webAccessSettings)
	mux.HandleFunc("POST /web-access-settings", s.saveWebAccessSettings)
}

func (s *Server) webAccessSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.ctl().WebAccessSettings())
}

func (s *Server) saveWebAccessSettings(w http.ResponseWriter, r *http.Request) {
	// A standing consent to be charged and the address queries go to are the
	// window's to set: no paired or network client decides them for the user.
	if !s.grants.at(r).providerEdit {
		refuse(w, http.StatusForbidden, "web_access.editing_disabled", "web search and fetch settings editing is not enabled on this server", nil)
		return
	}
	var patch control.WebAccessPatch
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&patch); err != nil || patch.Empty() {
		badBody(w)
		return
	}
	saved, err := s.ctl().SaveWebAccessSettings(patch)
	var unknown *config.UnknownWebBackendError
	switch {
	case errors.As(err, &unknown):
		refuse(w, http.StatusBadRequest, "web_access.unknown_backend", err.Error(), map[string]any{"backend": unknown.Name})
	case errors.Is(err, config.ErrBadSearxngURL):
		refuse(w, http.StatusBadRequest, "web_access.bad_searxng_url", err.Error(), nil)
	case err != nil:
		saveFailed(w, http.StatusInternalServerError, "web_access.save_failed", err)
	default:
		writeJSON(w, saved)
	}
}
