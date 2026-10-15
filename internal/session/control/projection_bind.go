package control

import (
	"fmt"
	"log/slog"
	"reasonix/internal/state/sessionstore"
	"time"

	"reasonix/internal/contract/event"
	"reasonix/internal/runtime/agent"
)

// bindExecutorProjection rebinds the agent's projection sidecar to path.
// loadSidecar=true loads an existing sidecar (resume/switch); false clears
// in-memory projection without deleting another session's .context.json.
func (c *Controller) bindExecutorProjection(path string, loadSidecar bool) {
	if c == nil || c.executor == nil {
		return
	}
	c.executor.BindSessionPath(path, loadSidecar)
	if loadSidecar {
		c.announceUnusedFold()
	}
}

// announceUnusedFold says why a fold saved beside the session was not reused: the
// full history goes to the model instead, and only the host knows which reason.
func (c *Controller) announceUnusedFold() {
	if c.sink == nil {
		return
	}
	switch c.executor.ProjectionDropped() {
	case agent.ProjectionDropHistory:
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Code: event.NoticeCodeFoldNotReusedHistory,
			Text: "A saved compaction of this conversation was not reused: the history it summarised has changed since. The full history is sent instead."})
	case agent.ProjectionDropLineage:
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Code: event.NoticeCodeFoldNotReusedLineage,
			Text: "A saved compaction of this conversation was made under another model or workspace and does not match its history. The full history is sent instead."})
	}
}

// maybeColdResumePrune records warm/cold/unknown only; it never rewrites
// history. announce=false keeps the state and drops the notice, for a rebuild
// that re-binds the session in place.
func (c *Controller) maybeColdResumePrune(path string, announce bool) {
	if c.executor == nil || path == "" {
		return
	}
	// Sidecar path is rebound in Resume; only refresh cache state here.
	m, ok, err := sessionstore.LoadBranchMeta(path)
	if err != nil || !ok || m.UpdatedAt.IsZero() {
		c.executor.SetCacheState(agent.CacheStateUnknown)
		slog.Debug("controller: resume cache state", "path", path, "cache_state", agent.CacheStateUnknown)
		return
	}
	last := m.UpdatedAt
	state := agent.CacheStateWarm
	if time.Since(last) >= c.cacheColdAfter() {
		state = agent.CacheStateCold
	}
	c.executor.SetCacheState(state)
	slog.Debug("controller: resume cache state", "path", path, "cache_state", state, "idle", time.Since(last).Round(time.Minute).String())
	if !announce || c.disableColdResumePrune || state != agent.CacheStateCold {
		return
	}
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
		"resumed after %s idle (provider cache likely expired) — full history kept; compaction deferred until context pressure",
		time.Since(last).Round(time.Minute))})
}
