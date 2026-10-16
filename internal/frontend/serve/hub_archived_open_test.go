package serve

import (
	"context"
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/provider"
	"reasonix/internal/state/sessionstore"
)

// Opening an archived conversation shows that conversation: it adds no second
// row to the list, and what the person archived stays archived until they say
// otherwise.
func TestOpeningAnArchivedSessionAddsNoRowAndKeepsItArchived(t *testing.T) {
	root := testenv.TempDir(t)
	writeServeModelConfig(t)
	closeSharedCatalogsOnCleanup(t)
	h := NewHub(HubOptions{ProviderResolver: homeResolver()})
	defer h.Shutdown()
	path := filepath.Join(SessionDirFor(root), "20261001-120000.jsonl")
	s := sessionstore.NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "first topic of the archived talk"})
	s.Add(provider.Message{Role: provider.RoleAssistant, Content: "an answer"})
	if err := s.SaveSnapshot(path); err != nil {
		t.Fatal(err)
	}
	if err := sessionstore.SetSessionArchived(path, true); err != nil {
		t.Fatal(err)
	}

	rt, err := h.Open(context.Background(), OpenRequest{Root: root, SessionPath: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { rt.Server.Controller().Close() })

	if got := rt.Server.Controller().SessionPath(); mustEval(t, got) != mustEval(t, path) {
		t.Fatalf("the pane is on %q, want the archived session %q", got, path)
	}
	rows := h.workspacePage(root, "", nil, treeWindow{scope: scopeEvery}, 0).Sessions
	if len(rows) != 1 {
		t.Fatalf("the list has %d rows after opening one archived session: %+v", len(rows), rows)
	}
	if !rows[0].Archived || rows[0].Path != path {
		t.Fatalf("the opened row is %+v, want the archived session itself", rows[0])
	}
	if active := h.workspacePage(root, "", nil, treeWindow{scope: scopeActive}, 0).Sessions; len(active) != 0 {
		t.Fatalf("the everyday list shows %+v for an archived session that is only open", active)
	}
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
