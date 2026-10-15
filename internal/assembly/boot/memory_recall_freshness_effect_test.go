package boot

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	"reasonix/internal/contract/event"
	"reasonix/internal/contract/provider"
	"reasonix/internal/state/memory"
)

// TestEffectRewrittenFactReachesTheProvider pins memory freshness at the
// provider boundary: after an external rewrite that preserves the file's byte
// length and modification time, the next request the executor actually sends
// must carry the new body. Compose-level assertions only prove what the
// controller would build; the model reads the provider request and nothing else.
func TestEffectRewrittenFactReachesTheProvider(t *testing.T) {
	isolateConfigHome(t)
	dir := robustTempDir(t)
	t.Chdir(dir)

	rec := &effectRecordingProvider{}
	provider.Register("boot-effect-memory-freshness", func(provider.Config) (provider.Provider, error) {
		return rec, nil
	})
	writeFile(t, dir, "reasonix.toml", `
default_model = "test-model"

[agent]
system_prompt = "BASE"

[[providers]]
name = "test-model"
kind = "boot-effect-memory-freshness"
model = "x"
`)
	approveWorkspace(t, dir)

	ctrl, err := Build(context.Background(), Options{Sink: event.Discard})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer ctrl.Close()

	path, err := ctrl.SaveMemory(memory.Memory{
		Name: "deploy-target", Description: "where the staging deploy goes",
		Keywords: "staging deploy target", Type: memory.TypeProject, Body: "old body marker",
	})
	if err != nil {
		t.Fatalf("SaveMemory: %v", err)
	}

	const query = "staging deploy target"
	run := func(text string) {
		t.Helper()
		if err := ctrl.Run(context.Background(), text); err != nil {
			t.Fatalf("Run(%q): %v", text, err)
		}
	}
	run("hello") // a save queues a note that suppresses recall for one turn
	run(query)
	if got := lastUserTurn(t, agentRequests(rec.requests())); !strings.Contains(got, "old body marker") {
		t.Fatalf("the fact never reached the provider, so the freshness claim under test is unproven:\n%s", got)
	}

	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// "new body marker" is exactly as long as "old body marker".
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "old body marker", "new body marker", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("the rewrite must be invisible to a stat for this test to bite: size %d→%d, mtime %v→%v",
			before.Size(), after.Size(), before.ModTime(), after.ModTime())
	}

	run(query)
	got := lastUserTurn(t, agentRequests(rec.requests()))
	if strings.Contains(got, "old body marker") {
		t.Fatalf("the provider is still sent the stale fact after an external rewrite:\n%s", got)
	}
	if !strings.Contains(got, "new body marker") {
		t.Fatalf("the refreshed fact never reached the provider boundary:\n%s", got)
	}
}

// lastUserTurn is the newest user message of the newest agent request — the
// turn the provider is about to be sent, not the transcript it already holds.
func lastUserTurn(t *testing.T, reqs []provider.Request) string {
	t.Helper()
	if len(reqs) == 0 {
		t.Fatal("no agent request reached the provider boundary")
	}
	req := reqs[len(reqs)-1]
	for _, msg := range slices.Backward(req.Messages) {
		if msg.Role == provider.RoleUser {
			return msg.Content
		}
	}
	t.Fatal("the newest agent request carried no user message")
	return ""
}
