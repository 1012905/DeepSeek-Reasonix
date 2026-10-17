package serve

import (
	"encoding/json"
	"errors"
	"net/http"

	"reasonix/internal/session/control"
)

const (
	codePeersDisabled  = "peer.disabled"
	codePeerNotFound   = "peer.not_found"
	codePeerAmbiguous  = "peer.ambiguous"
	codePeerSelf       = "peer.self"
	codePeerRefused    = "peer.refused"
	codePeerRateLimit  = "peer.rate_limited"
	codePeerEmpty      = "peer.empty"
	codePeerTooLarge   = "peer.too_large"
	codePeerNotHeld    = "peer.not_held"
	codePeerSendFailed = "peer.send_failed"
)

func (s *Server) registerPeerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /peers", s.peers)
	mux.HandleFunc("POST /peers/send", s.sendPeerMessage)
	mux.HandleFunc("POST /peers/held/{id}", s.resolvePeerMessage)
}

func (s *Server) peers(w http.ResponseWriter, _ *http.Request) {
	ctl := s.ctl()
	peers, held := ctl.Peers(), ctl.HeldPeerMessages()
	if peers == nil {
		peers = []control.PeerView{}
	}
	if held == nil {
		held = []control.PeerHeld{}
	}
	threads := ctl.PeerThreads()
	if threads == nil {
		threads = []control.PeerThread{}
	}
	writeJSON(w, map[string]any{"enabled": ctl.PeersEnabled(), "peers": peers, "held": held, "threads": threads})
}

// sendPeerMessage and resolvePeerMessage ride the provider-edit grant: a held
// message becomes input to a model, so a networked client must not decide for
// the machine's owner what another session is told.
func (s *Server) sendPeerMessage(w http.ResponseWriter, r *http.Request) {
	if !s.grants.at(r).providerEdit {
		refuse(w, http.StatusForbidden, "peer.editing_disabled", "messages between sessions are not enabled for this client", nil)
		return
	}
	var body struct {
		To   string `json:"to"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&body); err != nil {
		badBody(w)
		return
	}
	receipt, err := s.ctl().SendPeerMessage(body.To, body.Text)
	if err != nil {
		refusePeer(w, err)
		return
	}
	writeJSON(w, receipt)
}

func (s *Server) resolvePeerMessage(w http.ResponseWriter, r *http.Request) {
	if !s.grants.at(r).providerEdit {
		refuse(w, http.StatusForbidden, "peer.editing_disabled", "messages between sessions are not enabled for this client", nil)
		return
	}
	var body struct {
		Approve *bool `json:"approve"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil || body.Approve == nil {
		badBody(w)
		return
	}
	if err := s.ctl().ResolvePeerMessage(r.PathValue("id"), *body.Approve); err != nil {
		refusePeer(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func refusePeer(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, control.ErrPeersDisabled):
		refuse(w, http.StatusConflict, codePeersDisabled, "messages between sessions are not enabled", nil)
	case errors.Is(err, control.ErrPeerNotFound):
		refuse(w, http.StatusNotFound, codePeerNotFound, "no session on this machine answers to that name", nil)
	case errors.Is(err, control.ErrPeerAmbiguous):
		refuse(w, http.StatusConflict, codePeerAmbiguous, "more than one session answers to that name", nil)
	case errors.Is(err, control.ErrPeerSelf):
		refuse(w, http.StatusBadRequest, codePeerSelf, "a session cannot message itself", nil)
	case errors.Is(err, control.ErrPeerRefused):
		refuse(w, http.StatusConflict, codePeerRefused, "that session does not accept messages", nil)
	case errors.Is(err, control.ErrPeerRateLimited):
		refuse(w, http.StatusTooManyRequests, codePeerRateLimit, "too many messages sent in a short time", nil)
	case errors.Is(err, control.ErrPeerEmpty):
		refuse(w, http.StatusBadRequest, codePeerEmpty, "a message needs some text", nil)
	case errors.Is(err, control.ErrPeerTooLarge):
		refuse(w, http.StatusRequestEntityTooLarge, codePeerTooLarge, "that message is too long", nil)
	case errors.Is(err, control.ErrPeerNotHeld):
		refuse(w, http.StatusNotFound, codePeerNotHeld, "that message is no longer waiting", nil)
	default:
		saveFailed(w, http.StatusInternalServerError, codePeerSendFailed, err)
	}
}
