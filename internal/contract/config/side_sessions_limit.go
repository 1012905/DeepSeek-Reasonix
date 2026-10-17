package config

import "strings"

// SideSessionLimit is how many side conversations may be open at once; zero,
// the default, sets no limit. A negative value in the file reads as none.
func (c *Config) SideSessionLimit() int {
	if c == nil {
		return 0
	}
	return max(c.Agent.MaxSideSessions, 0)
}

// renderConcurrencyLimits writes the settings that bound how much runs at once.
func renderConcurrencyLimits(b *strings.Builder, c *Config, defaults *Config) {
	agentSetting(b, "max_subagent_depth", c.Agent.MaxSubagentDepth, defaults.Agent.MaxSubagentDepth, "nested subagent delegation depth; 1 restores the old single-layer boundary")
	agentSetting(b, "max_subagent_concurrency", c.Agent.MaxSubagentConcurrency, defaults.Agent.MaxSubagentConcurrency, "session-wide sub-agent concurrency (task/fleet/skills)")
	agentSetting(b, "max_parallel_writers", c.Agent.MaxParallelWriters, defaults.Agent.MaxParallelWriters, "concurrent writers with non-overlapping write_paths")
	agentSetting(b, "max_side_sessions", c.Agent.MaxSideSessions, defaults.Agent.MaxSideSessions, "side conversations open at once; 0 = no limit; user file only")
}
