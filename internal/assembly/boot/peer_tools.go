package boot

import (
	"slices"

	"reasonix/internal/contract/config"
	"reasonix/internal/contract/tool"
	"reasonix/internal/tools/builtin"
)

// addPeerTools registers the tools that let the model see and message other
// sessions, and only in a session that turned messages between sessions on, so
// the tool block of every other session is unchanged.
func addPeerTools(reg *tool.Registry, cfg *config.Config) *tool.PeerLink {
	if !cfg.PeersEnabled() {
		return nil
	}
	link := &tool.PeerLink{}
	for _, t := range builtin.PeerTools(link) {
		if len(cfg.Tools.Enabled) == 0 || slices.Contains(cfg.Tools.Enabled, t.Name()) {
			reg.Add(t)
		}
	}
	return link
}
