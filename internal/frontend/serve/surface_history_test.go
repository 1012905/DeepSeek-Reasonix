package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
	"reasonix/internal/contract/eventwire"
	"reasonix/internal/state/store"
)

type surfaceCtl struct {
	extensionCtl
	path     string
	surfaces []event.ExtensionSurfacePayload
}

func (c *surfaceCtl) SessionPath() string                                { return c.path }
func (c *surfaceCtl) ExtensionSurfaces() []event.ExtensionSurfacePayload { return c.surfaces }

func TestSurfaceMarkerNeverRecordsPluginContent(t *testing.T) {
	path := filepath.Join(testenv.TempDir(t), "session.jsonl")
	var log wireLog
	log.write(path, []byte(`{"kind":"extension_surface","extension":{"pluginId":"alpha","surfaceId":"form","kind":"form","form":{"message":"PRIVATE","fields":[{"key":"secret","kind":"input","default":"SECRET"}]}}}`))
	log.write(path, []byte(`{"kind":"extension_surface","extension":{"pluginId":"alpha","surfaceId":"toast","kind":"notification","notification":{"title":"DO NOT REPLAY"}}}`))
	data, err := os.ReadFile(store.SessionWireLog(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"PRIVATE", "SECRET", "default", "DO NOT REPLAY", "toast"} {
		if strings.Contains(string(data), private) {
			t.Fatalf("recorded content: %s", data)
		}
	}
	if !strings.Contains(string(data), event.SurfaceExpired) {
		t.Fatalf("no expiry identity: %s", data)
	}
}

func TestSurfaceRecoveryUsesLiveStateAndExpiresAfterRestart(t *testing.T) {
	path := filepath.Join(testenv.TempDir(t), "session.jsonl")
	var log wireLog
	log.write(path, []byte(`{"kind":"extension_surface","extension":{"pluginId":"alpha","surfaceId":"view","kind":"view","view":{"body":[{"kind":"text","value":"PRIVATE"}]}}}`))
	ctl := &surfaceCtl{path: path, surfaces: []event.ExtensionSurfacePayload{{PluginID: "alpha", SurfaceID: "view", Kind: "view", View: &event.ExtensionViewSurface{Body: []event.ExtensionViewNode{{Kind: "text", Value: "live"}}}}}}
	s := New(ctl, NewBroadcaster(), config.ServeConfig{})
	for _, live := range []bool{true, false} {
		if !live {
			ctl.surfaces = nil
		}
		req, _ := http.NewRequest(http.MethodGet, "/extensions/surfaces", nil)
		rec := httptest.NewRecorder()
		s.extensionSurfaces(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		raw := rec.Body.Bytes()
		var got []eventwire.ExtensionSurface
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("recovery: %s", raw)
		}
		if live && (got[0].View == nil || got[0].View.Body[0].Value != "live") {
			t.Fatalf("lost live view: %s", raw)
		}
		if !live && (got[0].Code != event.SurfaceExpired || got[0].View != nil) {
			t.Fatalf("old view stays live: %s", raw)
		}
	}
}
