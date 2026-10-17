package control

import (
	"context"
	"errors"
	"testing"

	"reasonix/internal/contract/agentcomposition"
	"reasonix/internal/runtime/agent/testutil"
	"reasonix/internal/state/sessioninbox"
	"reasonix/internal/state/sessionstore"
)

func testComposition(t *testing.T, c *Controller) {
	t.Helper()
	d := agentcomposition.Builtins()[1]
	b := &agentcomposition.Binding{Resolved: d, Version: agentcomposition.Version{Definition: d, Revision: agentcomposition.Revision(d)}}
	b.Seal()
	c.BindComposition(b)
	c.ConfigureCompositions(t.TempDir(), nil)
}

func TestCompositionSwitchAndFirstTurnShareAdmission(t *testing.T) {
	c, _, done := newOneTurnController(t, testutil.Turn{Text: "answer"})
	testComposition(t, c)
	entered, finish := make(chan struct{}), make(chan struct{})
	c.ConfigureCompositions(t.TempDir(), func(context.Context, string) error { close(entered); <-finish; return errors.New("candidate failed") })
	result := make(chan error, 1)
	go func() { result <- c.SelectAgentComposition(t.Context(), "builtin:research") }()
	<-entered
	if got := c.SubmitHTTPFormat("first prompt", ""); got != turnDroppedRotating {
		t.Fatalf("admission=%v", got)
	}
	close(finish)
	if err := <-result; err == nil {
		t.Fatal("build failure hidden")
	}
	if c.CompositionBinding().Started {
		t.Fatal("refused prompt locked old session")
	}
	c.Submit("accepted prompt")
	waitForDone(t, done)
	if err := c.SelectAgentComposition(t.Context(), "builtin:research"); !errors.Is(err, agentcomposition.ErrLocked) {
		t.Fatalf("started switch=%v", err)
	}
	if err := c.ClearSession(); err != nil {
		t.Fatal(err)
	}
	if err := c.SelectAgentComposition(t.Context(), "builtin:research"); !errors.Is(err, agentcomposition.ErrLocked) {
		t.Fatalf("clear unlocked binding: %v", err)
	}
}

func TestCompositionQueueAdmissionSurvivesRestartAndDeletion(t *testing.T) {
	c, _, _ := newOneTurnController(t)
	testComposition(t, c)
	if err := c.Snapshot(); err != nil {
		t.Fatal(err)
	}
	r, err := c.EnqueueInbox(InboxRequest{Submit: "queued work"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteInboxItem(r.ItemID); err != nil {
		t.Fatal(err)
	}
	loaded, err := sessionstore.LoadSession(c.SessionPath())
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Composition().Started {
		t.Fatal("queue acceptance was not durable")
	}
	if _, err := c.BeginCompositionSwitch(); !errors.Is(err, agentcomposition.ErrLocked) {
		t.Fatalf("queue deletion unlocked=%v", err)
	}
	c.Close()
	if _, err := c.EnqueueInbox(InboxRequest{Submit: "stale controller"}); err == nil || errors.Is(err, sessioninbox.ErrEmpty) {
		t.Fatalf("closed enqueue=%v", err)
	}
}
