package config

import "fmt"

// agentLimits are the three counts that bound how much delegation runs at once.
// A repo may not move any of them: lowering one is a stall, and raising the
// depth is unbounded cost, so the user's own file decides all three.
type agentLimits struct {
	concurrency, writers, depth int
}

func holdAgentLimits(c *Config) agentLimits {
	return agentLimits{c.Agent.MaxSubagentConcurrency, c.Agent.MaxParallelWriters, c.Agent.MaxSubagentDepth}
}

func (h heldScope) narrowAgentLimits(c *Config) {
	for _, l := range []struct {
		key  string
		now  *int
		held int
	}{
		{"agent.max_subagent_concurrency", &c.Agent.MaxSubagentConcurrency, h.limits.concurrency},
		{"agent.max_parallel_writers", &c.Agent.MaxParallelWriters, h.limits.writers},
		{"agent.max_subagent_depth", &c.Agent.MaxSubagentDepth, h.limits.depth},
	} {
		if *l.now != l.held {
			c.ignoreProject(l.key, fmt.Sprint(*l.now), ProjectUserOnly)
			*l.now = l.held
		}
	}
}
