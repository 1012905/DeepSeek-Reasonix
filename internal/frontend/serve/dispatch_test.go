package serve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/session/control"
	"reasonix/internal/state/sessionstore"
)

type dispatchFixture struct {
	t        *testing.T
	hub      *Hub
	peersDir string
	home     string
	opened   int
}

func newDispatchFixture(t *testing.T, userFile string) *dispatchFixture {
	t.Helper()
	home := testenv.TempDir(t)
	t.Setenv("REASONIX_HOME", home)
	if userFile != "" {
		if err := os.WriteFile(config.UserConfigPath(), []byte(userFile), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f := &dispatchFixture{t: t, hub: NewHub(HubOptions{}), peersDir: filepath.Join(home, "peers"), home: home}
	f.hub.dispatch.open = func(_ context.Context, req OpenRequest) (*Runtime, error) {
		f.opened++
		return f.adopt(req.Root, ""), nil
	}
	return f
}

// adopt puts a peers-enabled session of root into the hub; path "" leaves it
// without a file yet, as a pane the hub just opened is.
func (f *dispatchFixture) adopt(root, path string) *Runtime {
	f.t.Helper()
	ctrl := control.New(control.Options{
		WorkspaceRoot: root, SessionDir: SessionDirFor(root), SessionPath: path,
		Peers:   control.PeersOptions{Dir: f.peersDir},
		Posture: control.PostureEvidence{Home: f.home},
	})
	f.t.Cleanup(ctrl.Close)
	bc := NewBroadcaster()
	rt, err := f.hub.Adopt(New(ctrl, bc, config.ServeConfig{}), bc)
	if err != nil {
		f.t.Fatal(err)
	}
	return rt
}

func (f *dispatchFixture) project(name string, trusted bool) string {
	f.t.Helper()
	dir := filepath.Join(testenv.TempDir(f.t), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	rememberWorkspace(dir)
	if trusted {
		if err := config.NewProjectGrantStore(f.home).SetTrust(dir, config.WorkspaceTrusted); err != nil {
			f.t.Fatal(err)
		}
	}
	return dir
}

func (f *dispatchFixture) sender(root string) *control.Controller {
	f.t.Helper()
	path := sessionstore.NewSessionPath(SessionDirFor(root), "sender")
	rt := f.adopt(root, path)
	return rt.Server.Controller().(*control.Controller)
}

const dispatchOn = "[peers]\nenabled = true\ndispatch = true\n"

func TestDispatchOpensASessionInTheOtherProjectAndHoldsTheTaskThere(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	a, b := f.project("alpha", true), f.project("billing", true)
	sender := f.sender(a)

	got, err := sender.DispatchToProject(context.Background(), "billing", "rotate the key")
	if err != nil {
		t.Fatal(err)
	}
	if got.Project != b || got.Key == "" || got.ID == "" || f.opened != 1 {
		t.Fatalf("result = %+v, opened %d; want the billing project, a key, an id and one pane", got, f.opened)
	}
	var receiver *control.Controller
	for _, rt := range f.hub.localRuntimes() {
		if c, ok := rt.Server.Controller().(*control.Controller); ok && c.PeerKey() == got.Key {
			receiver = c
		}
	}
	if receiver == nil {
		t.Fatal("no pane answers to the returned key")
	}
	receiver.RefreshPeers()
	held := receiver.HeldPeerMessages()
	if len(held) != 1 || held[0].Body != "rotate the key" {
		t.Fatalf("held = %+v, want the task waiting for the receiver's user", held)
	}
	if !receiver.OpenedByDispatch() {
		t.Fatal("the opened session is not marked as opened by a dispatch")
	}

	again, err := sender.DispatchToProject(context.Background(), "billing", "and one more thing")
	if err != nil || again.Key != got.Key || f.opened != 1 {
		t.Fatalf("second dispatch = %+v, %v with %d panes opened; want the same session reused", again, err, f.opened)
	}
}

func TestDispatchRefusesWithTypedIdentities(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	a := f.project("alpha", true)
	f.project("same", true)
	f.project("same", true)
	f.project("stranger", false)
	sender := f.sender(a)
	ctx := context.Background()

	for name, tc := range map[string]struct {
		project string
		want    error
	}{
		"unknown":   {"nowhere", control.ErrDispatchUnknownProject},
		"ambiguous": {"same", control.ErrDispatchAmbiguous},
		"untrusted": {"stranger", control.ErrDispatchUntrusted},
		"outside":   {"/tmp", control.ErrDispatchUnknownProject},
	} {
		if _, err := sender.DispatchToProject(ctx, tc.project, "x"); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
	if got := len(f.hub.localRuntimes()); got != 1 {
		t.Fatalf("%d panes left open, want only the sender's (the untrusted one closed again)", got)
	}
}

func TestADispatchedSessionCannotDispatchFurther(t *testing.T) {
	f := newDispatchFixture(t, dispatchOn)
	a, _ := f.project("alpha", true), f.project("billing", true)
	sender := f.sender(a)
	if _, err := sender.DispatchToProject(context.Background(), "billing", "x"); err != nil {
		t.Fatal(err)
	}
	for _, rt := range f.hub.localRuntimes() {
		c := rt.Server.Controller().(*control.Controller)
		if c.OpenedByDispatch() {
			if _, err := c.DispatchToProject(context.Background(), "alpha", "back again"); !errors.Is(err, control.ErrDispatchNested) {
				t.Fatalf("nested dispatch = %v, want ErrDispatchNested", err)
			}
			return
		}
	}
	t.Fatal("no dispatched session found")
}

func TestDispatchIsOffUnlessTheUserFileSaysSo(t *testing.T) {
	f := newDispatchFixture(t, "[peers]\nenabled = true\n")
	f.project("billing", true)
	sender := f.sender(f.project("alpha", true))
	if _, err := sender.DispatchToProject(context.Background(), "billing", "x"); !errors.Is(err, control.ErrDispatchDisabled) {
		t.Fatalf("dispatch with the switch off = %v, want ErrDispatchDisabled", err)
	}
	if f.opened != 0 {
		t.Fatal("a pane was opened with dispatch off")
	}
}
