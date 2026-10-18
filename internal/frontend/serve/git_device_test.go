package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reasonix/internal/platform/gitpush"
	"reasonix/internal/session/control"
)

// gitCalls counts what reached the controller: a refused request must reach nothing.
type gitCalls struct {
	control.SessionAPI
	preview, push, discard int
}

func (g *gitCalls) GitPushPreview(_ context.Context, _ string) (gitpush.Plan, error) {
	g.preview++
	return gitpush.Plan{}, nil
}

func (g *gitCalls) GitPush(_ context.Context, _ control.PushRequest) (gitpush.Result, error) {
	g.push++
	return gitpush.Result{}, nil
}

func (g *gitCalls) DiscardChange(_ context.Context, _ control.DiscardRequest) error {
	g.discard++
	return nil
}

func gitRoute(g *gitCalls, method, path, body string, device bool) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	(&Server{ctrl: g}).registerGitRoutes(mux)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if device {
		req = req.WithContext(withDeviceReach(req.Context(), "phone-1", 1))
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAPairedDeviceCannotPreviewPushOrDiscardAndTheHostCan(t *testing.T) {
	routes := []struct {
		name, method, path, body string
		count                    func(*gitCalls) int
	}{
		{"preview a push", http.MethodGet, "/git/push", "", func(g *gitCalls) int { return g.preview }},
		{"push", http.MethodPost, "/git/push", `{"plan":{}}`, func(g *gitCalls) int { return g.push }},
		{"discard", http.MethodPost, "/changes/discard", `{"path":"a.go"}`, func(g *gitCalls) int { return g.discard }},
	}
	for _, rt := range routes {
		device := &gitCalls{}
		rec := gitRoute(device, rt.method, rt.path, rt.body, true)
		var reason Reason
		_ = json.Unmarshal(rec.Body.Bytes(), &reason)
		if rec.Code != http.StatusNotFound || reason.Code != codeDeviceHostOnly || rt.count(device) != 0 {
			t.Errorf("%s from a device: status %d code %q reached the controller %d times, want 404 %s and never", rt.name, rec.Code, reason.Code, rt.count(device), codeDeviceHostOnly)
		}
		host := &gitCalls{}
		rec = gitRoute(host, rt.method, rt.path, rt.body, false)
		if rec.Code == http.StatusNotFound || rt.count(host) != 1 {
			t.Errorf("%s from the host: status %d, reached the controller %d times, want it served once", rt.name, rec.Code, rt.count(host))
		}
	}
}
