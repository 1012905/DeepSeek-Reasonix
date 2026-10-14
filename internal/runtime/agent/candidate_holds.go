package agent

import "reasonix/internal/contract/provider"

// candidateHolds splits a candidate of total tokens into the parts a fold
// controls and the rest. The remainder is the recent tail, which is whatever
// the fold left in place after the summary.
func (a *contextWindow) candidateHolds(total, prefix int, kept []provider.Message, summary string) candidateHolds {
	h := candidateHolds{
		Prefix:  prefix,
		Kept:    a.estimatedPromptTokens(kept),
		Summary: a.estimatedPromptTokens([]provider.Message{{Role: provider.RoleUser, Content: summary}}),
	}
	h.Tail = max(total-h.Prefix-h.Kept-h.Summary, 0)
	return h
}

// acceptAttributed is acceptCheckpointCandidate whose size refusals name what
// the candidate is made of; the split is only worked out when one refuses.
func (a *contextWindow) acceptAttributed(trigger string, scope compactionScope, sourceTokens, candidateTokens, fixedPrefixTokens int, kept []provider.Message, summary string) error {
	err := a.acceptCheckpointCandidate(trigger, scope, sourceTokens, candidateTokens, fixedPrefixTokens)
	if err == nil {
		return nil
	}
	return attribute(err, a.candidateHolds(candidateTokens, fixedPrefixTokens, kept, summary))
}
