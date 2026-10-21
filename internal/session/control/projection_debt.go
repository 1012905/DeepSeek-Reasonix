// projection_debt.go — what the model has of one block that is delivered once.
package control

// projectionDebt answers "is this block owed" from the model-visible history,
// never from what a turn was handed: a refusal, a fold, a rewind or a session
// rotation all change the answer by themselves. newest reports the most recent
// block of a tag the model still reads (agent.NewestStandingBlock); nil means
// no model is attached, so every block is owed.
type projectionDebt struct {
	newest func(string) (string, bool)
}

// owed returns the projection unless the model's newest block of this tag is
// already those exact bytes. An older copy does not count: a later block of the
// same tag replaced it, and the caller may be owing those bytes again.
func (d *projectionDebt) owed(tag, projection string) string {
	if projection == "" {
		return ""
	}
	if d.newest != nil {
		if got, ok := d.newest(tag); ok && got == wrapTurnBlock(tag, projection) {
			return ""
		}
	}
	return projection
}

// bindProjectionSeer points every once-delivered block at the same model-visible
// history. New calls it once, before a turn can read a debt; a controller with
// no executor leaves every seer nil and owes every block, because no model has
// been told anything.
func (c *Controller) bindProjectionSeer(newest func(string) (string, bool), notes func(string) bool) {
	if newest == nil {
		return
	}
	c.date.debt.newest = newest
	c.scratch.debt.newest = newest
	c.skills.catalog.newest = newest
	c.memory.instructions.newest = newest
	c.rules.always.newest = newest
	c.rules.scoped.newest = newest
	c.deliveryNotes(notes)
}

// deliveryNotes binds the one-time facts a turn also carries. They are not
// standing blocks: a note is delivered when the model can read it anywhere.
func (c *Controller) deliveryNotes(notes func(string) bool) {
	if notes == nil {
		return
	}
	c.mcp.promptFailures.sees = notes
	c.refused.sees = notes
}

// visible reports whether the model still reads any block with this tag. It
// answers "is anything of this kind in view", where owed asks about the exact
// bytes this turn would render: a caller replacing a block needs the first
// question, and can then owe the replacement.
func (d *projectionDebt) visible(tag string) bool {
	if d.newest == nil || tag == "" {
		return false
	}
	_, ok := d.newest(tag)
	return ok
}
