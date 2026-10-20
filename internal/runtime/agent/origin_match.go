package agent

import "reasonix/internal/contract/provider"

// noteOriginMatch counts a call that is not read-only and whose arguments carry,
// verbatim, text an external result gave this task. Nothing is decided on it: the
// number goes to the turn's audit, so a later gate can be set from measured rates.
func (a *Agent) noteOriginMatch(plan *toolCallPlan) {
	if plan == nil || plan.readOnly || a.task.origins == nil {
		return
	}
	if len(a.task.origins.match(plan.permArgs, a.userTexts)) > 0 {
		a.task.origins.noteMatch()
	}
}

// userTexts is what the user wrote in this session, for telling a string they
// typed from one they only saw.
func (a *Agent) userTexts() []string {
	if a.sess.conversation == nil {
		return nil
	}
	var out []string
	for _, m := range a.sess.conversation.Snapshot() {
		if m.Role == provider.RoleUser {
			out = append(out, m.Content)
		}
	}
	return out
}
