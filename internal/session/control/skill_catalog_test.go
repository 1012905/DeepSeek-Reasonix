package control

import (
	"strings"
	"testing"

	"reasonix/internal/contract/event"
	"reasonix/internal/ext/skill"
)

// The listing is delivered once, on a user turn. A fold can summarise that turn
// away, and standing state that is only ever said once does not survive that —
// so a completed compaction has to owe it again.
func TestSkillCatalogIsOwedAgainAfterAFold(t *testing.T) {
	c := New(Options{Skills: []skill.Skill{{Name: "alpha", Description: "does the alpha thing"}}})

	first := c.Compose("hello")
	c.settleTurnProjections()
	if !strings.Contains(first, "<available-skills>") || !strings.Contains(first, "alpha") {
		t.Fatalf("the first turn did not carry the listing: %q", first)
	}
	second := c.Compose("again")
	c.settleTurnProjections()
	if strings.Contains(second, "<available-skills>") {
		t.Fatalf("the listing was delivered twice without anything changing: %q", second)
	}

	c.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{Messages: 4}})

	if after := c.Compose("after the fold"); !strings.Contains(after, "<available-skills>") || !strings.Contains(after, "alpha") {
		t.Fatalf("the fold left the session without a listing: %q", after)
	}
}

// A turn that composed and never reached the runner delivered nothing, so the
// debt stands. The settle point is the runner hand-off precisely because a hook
// or an extension can still refuse the turn after it composed.
func TestCatalogDebtStandsUntilTheTurnCarriesIt(t *testing.T) {
	c := New(Options{Skills: []skill.Skill{{Name: "alpha", Description: "does the alpha thing"}}})

	if first := c.Compose("hello"); !strings.Contains(first, "alpha") {
		t.Fatalf("the first turn did not carry the listing: %q", first)
	}
	abandoned := c.Compose("blocked before it ran")
	if !strings.Contains(abandoned, "alpha") {
		t.Fatalf("a composed-but-undelivered turn cleared the debt: %q", abandoned)
	}
	c.settleTurnProjections()
	if delivered := c.Compose("and now"); strings.Contains(delivered, "<available-skills>") {
		t.Fatalf("the listing was re-sent after a turn carried it: %q", delivered)
	}
}

// A fold that folded nothing — a truncated or refused digest — left the turn
// carrying the listing verbatim, so the listing is not owed again: re-sending it
// after every failed fold only grows the prompt the next attempt has to fold.
func TestSkillCatalogIsNotOwedAgainAfterAFoldThatFoldedNothing(t *testing.T) {
	c := New(Options{Skills: []skill.Skill{{Name: "alpha", Description: "does the alpha thing"}}})
	if first := c.Compose("hello"); !strings.Contains(first, "<available-skills>") {
		t.Fatalf("the first turn did not carry the listing: %q", first)
	}
	c.settleTurnProjections()

	c.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{Code: "summary_truncated"}})

	if after := c.Compose("after the failed fold"); strings.Contains(after, "<available-skills>") {
		t.Fatalf("a fold that folded nothing re-sent the listing: %q", after)
	}
}
