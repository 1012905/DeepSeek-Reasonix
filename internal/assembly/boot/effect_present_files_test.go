package boot

import (
	"context"
	"reflect"
	"testing"
	"time"

	"reasonix/internal/contract/event"
	"reasonix/internal/contract/eventwire"
	"reasonix/internal/contract/provider"
	"reasonix/internal/runtime/agent/testutil"
	"reasonix/internal/session/control"
)

func TestEffectPresentFilesSurvivesTheRealToolPipeline(t *testing.T) {
	for _, proxy := range []bool{false, true} {
		name := "direct"
		if proxy {
			name = "capability"
		}
		t.Run(name, func(t *testing.T) { presentFilesEffect(t, proxy) })
	}
}

func presentFilesEffect(t *testing.T, proxy bool) {
	t.Helper()
	isolateConfigHome(t)
	dir := robustTempDir(t)
	t.Chdir(dir)
	registerBootTokenProfileTestProvider()
	call := provider.ToolCall{ID: "present", Name: "present_files", Arguments: `{"files":[{"path":"report.md","description":"Report"}]}`}
	if proxy {
		call.Name = "use_capability"
		call.Arguments = `{"action":"call","capability_id":"tool:present_files","arguments":{"files":[{"path":"report.md","description":"Report"}]}}`
	}
	prov := testutil.NewMock("present-files",
		testutil.Turn{ToolCalls: []provider.ToolCall{call}},
		testutil.Turn{Text: "Done."},
	)
	setBootTokenProfileTestProvider(t, prov)
	writeFile(t, dir, "reasonix.toml", "default_model = \"test-model\"\n[[providers]]\nname = \"test-model\"\nkind = \"boot-token-profile-test\"\nmodel = \"x\"\n")
	writeFile(t, dir, "report.md", "# Report")
	approveWorkspace(t, dir)
	events := make(chan event.Event, 1000)
	ctrl, err := Build(context.Background(), Options{WorkspaceRoot: dir, Sink: event.FuncSink(func(e event.Event) { events <- e })})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ctrl.Close)
	if err := ctrl.NewSession(); err != nil {
		t.Fatal(err)
	}
	ctrl.SetToolApprovalMode(control.ToolApprovalYolo)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ctrl.Run(ctx, "Present the existing report.md file."); err != nil {
		t.Fatal(err)
	}
	found := false
	for len(events) > 0 {
		e := <-events
		if e.Kind == event.ToolResult && e.Tool.ID == "present" {
			wire := eventwire.ToWire(e)
			if wire.Tool == nil || len(wire.Tool.Deliveries) != 1 {
				t.Fatalf("delivery did not reach UI: %+v, output %q", e.Tool, e.Tool.Output)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("present tool result missing")
	}
	found = false
	for _, message := range ctrl.History() {
		for _, call := range message.ToolCalls {
			if call.ID == "present" && call.CapabilityID == "tool:present_files" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("host-confirmed identity was not persisted")
	}
	requests := prov.Requests()
	if len(requests) < 2 || !reflect.DeepEqual(requests[0].Tools, requests[1].Tools) {
		t.Fatal("delivery changed the tool schema within the turn")
	}
}
