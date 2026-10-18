package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"reasonix/internal/contract/config"
)

type sample struct {
	A bool   `toml:"a"`
	B string `toml:"b"`
}

func TestAnEntryWithoutAClassOrTimingFailsRegistration(t *testing.T) {
	root := reflect.TypeFor[sample]()
	for name, rows := range map[string][]Entry{
		"no class":  {{ID: "a", Effect: Immediate}, {ID: "b", Class: Preference, Effect: Immediate}},
		"no timing": {{ID: "a", Class: Preference}, {ID: "b", Class: Preference, Effect: Immediate}},
	} {
		if err := register(rows, root); err == nil {
			t.Errorf("%s: registered", name)
		}
	}
}

func TestAKeyNoRowDeclaresFailsRegistration(t *testing.T) {
	err := register([]Entry{{ID: "a", Class: Preference, Effect: Immediate}}, reflect.TypeFor[sample]())
	if CodeOf(err) != codeUndeclared || !strings.Contains(err.Error(), "b") {
		t.Fatalf("err = %v, want the undeclared key named", err)
	}
}

func TestARowForNoKeyFailsRegistration(t *testing.T) {
	rows := []Entry{{ID: "a", Class: Preference, Effect: Immediate}, {ID: "b", Class: Preference, Effect: Immediate}, {ID: "c", Class: Preference, Effect: Immediate}}
	if err := register(rows, reflect.TypeFor[sample]()); err == nil {
		t.Fatal("a declared key the config does not have was accepted")
	}
}

func TestTheDeclarationsCoverTheUserConfigExactly(t *testing.T) {
	if err := register(table, reflect.TypeFor[config.Config]()); err != nil {
		t.Fatal(err)
	}
	if len(All()) != len(fields) {
		t.Fatalf("declared %d, config has %d", len(All()), len(fields))
	}
}

func TestBoundaryKeysAreGuarded(t *testing.T) {
	for _, e := range All() {
		for _, prefix := range []string{"permissions.", "sandbox.", "network.", "environment."} {
			if strings.HasPrefix(e.ID, prefix) && e.Class != Guarded {
				t.Errorf("%s is under %s and must be Guarded", e.ID, prefix)
			}
		}
	}
	for _, id := range []string{
		"desktop.default_tool_approval_mode", "agent.plan_approval", "agent.host_verification", "agent.subagent_claims",
		"agent.max_steps", "agent.max_subagent_depth", "agent.max_subagent_concurrency", "agent.max_parallel_writers",
		"agent.task_token_budget", "agent.guardian_model", "agent.system_prompt", "agent.screen_external_content",
		"tools.shell.path", "tools.enabled", "statusline.command", "skills.paths", "providers", "plugins",
		"peers.release_projects", "serve.token", "serve.auth_mode", "secrets.protect_sensitive_files", "storage",
	} {
		if e, ok := Lookup(id); !ok || e.Class != Guarded {
			t.Errorf("%s = %+v, want Guarded", id, e)
		}
	}
}

func TestDispatchIsElevatedAndItsProjectListIsNot(t *testing.T) {
	for id, want := range map[string]Class{"peers.enabled": Elevated, "peers.dispatch": Elevated, "peers.release_projects": Guarded} {
		if e, _ := Lookup(id); e.Class != want {
			t.Errorf("%s class = %v, want %v", id, e.Class, want)
		}
	}
}

func TestASecretIsNeverRead(t *testing.T) {
	cfg := &config.Config{}
	cfg.Serve.Token = "hunter2"
	got, err := Read(cfg, "serve.token")
	if CodeOf(err) != CodeSecret || got != nil {
		t.Fatalf("got %v, %v; want the secret refusal and no value", got, err)
	}
}

func TestAGuardedKeyIsRefusedAndLeftAlone(t *testing.T) {
	cfg := &config.Config{}
	cfg.Permissions.Mode = "ask"
	_, _, err := Write(cfg, "permissions.mode", json.RawMessage(`"yolo"`))
	if CodeOf(err) != CodeGuarded || cfg.Permissions.Mode != "ask" {
		t.Fatalf("err = %v, mode = %q; want refused and unchanged", err, cfg.Permissions.Mode)
	}
}

func TestWriteReturnsWhatWasThereAndWhatIsNow(t *testing.T) {
	cfg := &config.Config{}
	cfg.UI.ShowReceipt = true
	from, to, err := Write(cfg, "ui.show_receipt", json.RawMessage(`false`))
	if err != nil || from != true || to != false || cfg.UI.ShowReceipt {
		t.Fatalf("from=%v to=%v err=%v show=%v", from, to, err, cfg.UI.ShowReceipt)
	}
	from, to, err = Write(cfg, "agent.max_queued_inputs", json.RawMessage(`7`))
	if err != nil || from != 0 || to != 7 || cfg.Agent.MaxQueuedInputs != 7 {
		t.Fatalf("from=%v to=%v err=%v", from, to, err)
	}
}

func TestAValueOfTheWrongTypeIsRefusedAndLeftAlone(t *testing.T) {
	cfg := &config.Config{}
	for _, raw := range []string{`"yes"`, `null`, `1.5`, `[]`} {
		before := *cfg
		_, _, err := Write(cfg, "ui.show_receipt", json.RawMessage(raw))
		if CodeOf(err) != CodeValue || !reflect.DeepEqual(before.UI, cfg.UI) {
			t.Errorf("%s: err = %v", raw, err)
		}
	}
}

func TestAListOfStringsAndNothingStructuredIsWritable(t *testing.T) {
	cfg := &config.Config{}
	if _, to, err := Write(cfg, "skills.disabled_skills", json.RawMessage(`["a","b"]`)); err != nil || !reflect.DeepEqual(to, []string{"a", "b"}) {
		t.Fatalf("to=%v err=%v", to, err)
	}
	if _, _, err := Write(cfg, "agent.role_efforts", json.RawMessage(`{"x":"high"}`)); CodeOf(err) != CodeShape {
		t.Fatalf("a table was written: %v", err)
	}
}

func TestAnUnknownKeyIsRefusedByCode(t *testing.T) {
	if _, _, err := Write(&config.Config{}, "ui.nope", json.RawMessage(`1`)); CodeOf(err) != CodeUnknown {
		t.Fatalf("err = %v", err)
	}
}

func TestShapesAreReadFromTheConfigType(t *testing.T) {
	for id, want := range map[string]Shape{
		"ui.show_receipt": ShapeBool, "agent.max_queued_inputs": ShapeInt, "agent.temperature": ShapeFloat,
		"ui.theme": ShapeString, "skills.disabled_skills": ShapeStringList, "agent.role_efforts": ShapeStructured,
	} {
		if got := ShapeOf(id); got != want {
			t.Errorf("%s shape = %v, want %v", id, got, want)
		}
	}
}
