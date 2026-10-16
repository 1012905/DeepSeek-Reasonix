package boot

import (
	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
)

// peersOptions opens a session to messages from other sessions only when the
// user file asked for it; the directory is the one the sandbox already denies
// to confined commands.
func peersOptions(cfg *config.Config) control.PeersOptions {
	if !cfg.PeersEnabled() {
		return control.PeersOptions{}
	}
	return control.PeersOptions{Dir: cfg.Roots().PeersDir()}
}
