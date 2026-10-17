package plugin

import (
	"encoding/json"
	"testing"
)

func TestCompositionMCPContractBlocksRefreshCallsAndReconnectDrift(t *testing.T) {
	server, srv := newListChangedServer(t, true, "alpha")
	h, _ := connectListChanged(t, srv, Spec{Authorized: true})
	tools, err := h.ToolsFor(t.Context(), "live")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := h.PinCompositionTools("live", "")
	if err != nil || len(hash) != 64 {
		t.Fatalf("pin=%s %v", hash, err)
	}
	if _, err := h.PinCompositionTools("live", hash); err != nil {
		t.Fatal(err)
	}
	if _, err := h.PinCompositionTools("live", "different"); err == nil {
		t.Fatal("mismatched contract accepted")
	}
	if _, err := h.PinCompositionTools("missing", ""); err == nil {
		t.Fatal("missing server accepted")
	}
	if h.CompositionContractError() != nil {
		t.Fatal("valid contract blocked")
	}
	if _, err := tools[0].Execute(t.Context(), json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	before := server.calls.Load()
	server.announce(0, "alpha", "beta")
	if _, _, err := h.lookupClient("live").refreshToolCatalog(t.Context()); err == nil {
		t.Fatal("catalog drift published")
	}
	if h.CompositionContractError() == nil {
		t.Fatal("contract drift not reported")
	}
	if _, err := tools[0].Execute(t.Context(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("stale adapter executed after drift")
	}
	if server.calls.Load() != before {
		t.Fatal("blocked call reached server")
	}
	h.Remove("live")
	if _, err := h.EnsureConnected(t.Context(), Spec{Name: "live", Type: "http", URL: srv.URL, Authorized: true}); err == nil {
		t.Fatal("reconnect bypassed binding")
	}
}
