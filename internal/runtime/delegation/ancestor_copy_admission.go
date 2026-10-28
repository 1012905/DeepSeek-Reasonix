package delegation

import (
	"fmt"
	"strings"
)

type ancestorCopyKey struct {
	parent string
	source string
}

// A copied run has no durable metadata until its caller publishes it. Reserve
// the target-parent/source pair before looking for an existing copy, and keep
// that reservation for the returned run's ordinary lifetime. The ref lock alone
// cannot protect this gap because each preparation would mint a different ref.
func (s *SubagentStore) prepareContinueFromAncestor(ref string, spec SubagentSpec) (*SubagentRun, error) {
	source, err := s.nearestLineageSource(ref, spec)
	if err != nil {
		return nil, err
	}
	releaseCopy, err := s.reserveAncestorCopy(ancestorCopyKey{
		parent: strings.TrimSpace(spec.ParentSession), source: source,
	})
	if err != nil {
		return nil, err
	}
	run, err := s.prepareContinueFromSource(source, spec)
	if err != nil {
		releaseCopy()
		return nil, err
	}
	releaseRun := run.release
	run.release = func() {
		if releaseRun != nil {
			releaseRun()
		}
		releaseCopy()
	}
	return run, nil
}

func (s *SubagentStore) reserveAncestorCopy(key ancestorCopyKey) (func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.copying[key] {
		return nil, fmt.Errorf("subagent reference %q is already being continued in parent session %q; retry after it finishes", key.source, key.parent)
	}
	if s.copying == nil {
		s.copying = make(map[ancestorCopyKey]bool)
	}
	s.copying[key] = true
	return func() {
		s.mu.Lock()
		delete(s.copying, key)
		s.mu.Unlock()
	}, nil
}
