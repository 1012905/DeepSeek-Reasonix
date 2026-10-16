package control

import (
	"errors"
	"fmt"

	"reasonix/internal/contract/tool"
)

// peerDirectory is the controller as the peer tools see it. A refusal reaches
// the model as its identity first, so it can tell "no such session" from "slow
// down" without reading a sentence.
type peerDirectory struct{ c *Controller }

func (d peerDirectory) Peers() []tool.PeerView {
	var out []tool.PeerView
	for _, p := range d.c.Peers() {
		out = append(out, tool.PeerView{Key: p.Key, Title: p.Title, Project: p.Project})
	}
	return out
}

func (d peerDirectory) SendPeerMessage(to, text string) (tool.PeerReceipt, error) {
	r, err := d.c.SendPeerMessage(to, text)
	if err != nil {
		return tool.PeerReceipt{}, fmt.Errorf("%s: %w", peerErrorCode(err), err)
	}
	return tool.PeerReceipt{ID: r.ID, To: tool.PeerView{Key: r.To.Key, Title: r.To.Title, Project: r.To.Project}}, nil
}

func peerErrorCode(err error) string {
	for code, sentinel := range map[string]error{
		"peer.disabled": ErrPeersDisabled, "peer.not_found": ErrPeerNotFound, "peer.ambiguous": ErrPeerAmbiguous,
		"peer.self": ErrPeerSelf, "peer.refused": ErrPeerRefused, "peer.rate_limited": ErrPeerRateLimited,
		"peer.empty": ErrPeerEmpty, "peer.too_large": ErrPeerTooLarge,
	} {
		if errors.Is(err, sentinel) {
			return code
		}
	}
	return "peer.send_failed"
}
