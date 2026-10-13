package configbackup

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"reasonix/internal/ext/hook"
)

func TestHookBackupVariantsRoundTrip(t *testing.T) {
	machine(t, "")
	base := hook.HookConfig{Match: "edit_file", Command: "echo check", Cwd: "/tmp/project-a"}
	directory, timeout, description := base, base, base
	directory.Cwd = "/tmp/project-b"
	timeout.Timeout = 1234
	description.Description = "another label"
	want := []hook.HookConfig{base, directory, timeout, description}
	must(t, hook.Save(hook.ScopeGlobal, "", hook.Settings{Hooks: map[hook.Event][]hook.HookConfig{hook.PostToolUse: want}}))
	s := collectAll(t, CategoryAutomation)
	sealed, err := Seal(s, passphrase)
	must(t, err)
	s, err = Open(sealed, passphrase)
	must(t, err)
	if len(s.Items) != len(want) {
		t.Fatalf("collected %d hooks, want %d", len(s.Items), len(want))
	}
	machine(t, "")
	for range 2 {
		p := NewPlanner()
		plan, err := p.Preview(s)
		must(t, err)
		var ids []string
		for _, it := range plan.Items {
			ids = append(ids, it.ID)
		}
		result, err := p.Apply(ApplyRequest{PlanID: plan.ID, Items: ids, Consented: ids})
		must(t, err)
		if len(result.Failed) != 0 {
			t.Fatalf("restore failed: %+v", result)
		}
		got, err := globalHooks()
		must(t, err)
		raw, err := json.Marshal(got.Hooks[hook.PostToolUse])
		must(t, err)
		wantRaw, err := json.Marshal(want)
		must(t, err)
		if string(raw) != string(wantRaw) {
			t.Fatalf("restored hooks = %s, want %s", raw, wantRaw)
		}
	}
	plan, err := NewPlanner().Preview(s)
	must(t, err)
	for _, it := range plan.Items {
		if it.Status != StatusSame {
			t.Fatalf("restored hook is not the same: %+v", it)
		}
	}
	for _, detail := range []string{"timeout: 1234 ms", "description: another label"} {
		if !slices.ContainsFunc(plan.Items, func(row PlanItem) bool { return slices.Contains(row.Details, detail) }) {
			t.Fatalf("preview omits %q", detail)
		}
	}
}

func TestHookBackupDeduplicatesIdenticalHooks(t *testing.T) {
	machine(t, "")
	h := hook.HookConfig{Command: "echo check"}
	must(t, hook.Save(hook.ScopeGlobal, "", hook.Settings{Hooks: map[hook.Event][]hook.HookConfig{hook.PostToolUse: {h, h}}}))
	loaded := hook.Load(hook.LoadOptions{Scopes: []hook.Scope{hook.ScopeGlobal}})
	calls := 0
	hook.Run(context.Background(), hook.Payload{Event: hook.PostToolUse}, loaded, func(context.Context, hook.SpawnInput) hook.SpawnResult {
		calls++
		return hook.SpawnResult{}
	})
	if calls != 2 {
		t.Fatalf("configured duplicate hooks ran %d times, want 2", calls)
	}
	s := collectAll(t, CategoryAutomation)
	must(t, s.validate())
	if len(s.Items) != 1 {
		t.Fatalf("identical hooks produced %d restore items", len(s.Items))
	}
}

func TestLegacyHookBackupPreviewAndImport(t *testing.T) {
	machine(t, "")
	h := hook.HookConfig{Match: "edit_file", Command: "echo check", Cwd: " /tmp/project-a ", Description: " check files ", Timeout: 1234}
	must(t, hook.Save(hook.ScopeGlobal, "", hook.Settings{Hooks: map[hook.Event][]hook.HookConfig{hook.PostToolUse: {h}}}))
	s := &Snapshot{Format: 1, Categories: []Category{CategoryAutomation}, Items: []Item{
		fileItem(t, KindHook, "PostToolUse#d0b85885ddd6", hookData{Event: string(hook.PostToolUse), Hook: h}),
	}}
	sealed, err := Seal(s, passphrase)
	must(t, err)
	s, err = Open(sealed, passphrase)
	must(t, err)
	for range 2 {
		p := NewPlanner()
		plan, err := p.Preview(s)
		must(t, err)
		row := planRow(t, plan, s.Items[0].ID)
		if row.Status != StatusSame {
			t.Fatalf("legacy backup misidentified: %+v", row)
		}
		result, err := p.Apply(ApplyRequest{PlanID: plan.ID, Items: []string{row.ID}, Consented: []string{row.ID}})
		must(t, err)
		if len(result.Failed) != 0 {
			t.Fatalf("legacy restore failed: %+v", result)
		}
		got, err := globalHooks()
		must(t, err)
		if len(got.Hooks[hook.PostToolUse]) != 1 {
			t.Fatalf("legacy restore duplicated hook: %+v", got)
		}
	}
}
