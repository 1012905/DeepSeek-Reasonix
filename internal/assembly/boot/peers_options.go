package boot

import (
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/tool"
	"reasonix/internal/session/control"
)

// peersOptions opens a session to messages from other sessions only when the
// user file asked for it; the directory is the one the sandbox already denies
// to confined commands. The link lets the peer tools registered with the same
// switch reach the controller once it exists.
func peersOptions(cfg *config.Config, link *tool.PeerLink) control.PeersOptions {
	if !cfg.PeersEnabled() {
		return control.PeersOptions{}
	}
	return control.PeersOptions{Dir: cfg.Roots().PeersDir(), Link: link}
}
