package config

import (
	"fmt"
	"slices"
)

func clonePeers(p PeersConfig) PeersConfig {
	p.ReleaseProjects = slices.Clone(p.ReleaseProjects)
	return p
}

// narrowPeers puts back every messaging setting the project file touched: each
// one widens what a session of this user may send or accept, so only the user's
// own file decides.
func (h heldScope) narrowPeers(c *Config) {
	if c.Peers.Enabled != h.peers.Enabled {
		c.ignoreProject("peers.enabled", fmt.Sprintf("%t", c.Peers.Enabled), ProjectUserOnly)
	}
	if c.Peers.Dispatch != h.peers.Dispatch {
		c.ignoreProject("peers.dispatch", fmt.Sprintf("%t", c.Peers.Dispatch), ProjectUserOnly)
	}
	if !slices.Equal(c.Peers.ReleaseProjects, h.peers.ReleaseProjects) {
		c.ignoreProject("peers.release_projects", fmt.Sprint(c.Peers.ReleaseProjects), ProjectUserOnly)
	}
	c.Peers = clonePeers(h.peers)
}
