package boot

// A server that answers a tool call with a task must look, to the model, like a
// tool that took a while: the result arrives as the tool result, and nothing of
// the task protocol (its handle, its method names) is model-visible.

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"reasonix/internal/contract/event"
	"reasonix/internal/contract/provider"
)

const taskHandle = "task-handle-7f3a9c"

type taskRecorder struct {
	mu      sync.Mutex
	methods []string
}

func (r *taskRecorder) saw(method string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, m := range r.methods {
		if m == method {
			n++
		}
	}
	return n
}

func modernTaskServer(t *testing.T) (*httptest.Server, *taskRecorder) {
	t.Helper()
	rec := &taskRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     *int           `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		rec.mu.Lock()
		rec.methods = append(rec.methods, req.Method)
		rec.mu.Unlock()
		stamps := map[string]any{"createdAt": "2026-10-15T00:00:00Z", "lastUpdatedAt": "2026-10-15T00:00:01Z", "ttlMs": 60000, "pollIntervalMs": 1}
		var result map[string]any
		switch req.Method {
		case "server/discover":
			result = map[string]any{"supportedVersions": []string{"2026-07-28"}, "capabilities": map[string]any{
				"tools": map[string]any{}, "extensions": map[string]any{"io.modelcontextprotocol/tasks": map[string]any{}}}}
		case "tools/list":
			result = map[string]any{"ttlMs": 0, "tools": []map[string]any{{"name": "job", "description": "A long job.", "inputSchema": map[string]any{"type": "object"}}}}
		case "tools/call":
			result = map[string]any{"resultType": "task", "taskId": taskHandle, "status": "working"}
		case "tasks/get":
			result = map[string]any{"resultType": "complete", "taskId": taskHandle, "status": "completed",
				"result": map[string]any{"content": []map[string]any{{"type": "text", "text": "the long job finished"}}}}
		default:
			result = map[string]any{"resultType": "complete"}
		}
		if req.Method == "tools/call" || req.Method == "tasks/get" {
			maps.Copy(result, stamps)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": *req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestEffectMCPTaskReachesTheModelAsAnOrdinaryToolResult(t *testing.T) {
	home := isolateConfigHome(t)
	reasonixHome := filepath.Join(home, ".reasonix")
	t.Setenv("REASONIX_HOME", reasonixHome)
	workspace := robustTempDir(t)
	t.Chdir(workspace)

	srv, server := modernTaskServer(t)
	rec := &listChangedProvider{calls: map[int]string{0: capabilityCall("mcp-tool:live/job")}}
	provider.Register("boot-mcp-task", func(provider.Config) (provider.Provider, error) { return rec, nil })
	block := fmt.Sprintf("\n[[plugins]]\nname = \"live\"\ntype = \"http\"\nurl = %q\nload = \"always\"\n", srv.URL)
	writeFile(t, workspace, "reasonix.toml", listChangedConfig("boot-mcp-task", block))
	approveWorkspace(t, workspace)
	approveProjectServer(t, workspace, "live")

	ctrl, err := Build(t.Context(), Options{Sink: event.Discard, Home: reasonixHome, WorkspaceRoot: workspace})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Cleanup(ctrl.Close)
	if err := ctrl.Run(t.Context(), "run the long job"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := toolResults(rec.requests())["probe-0"]; !strings.Contains(got, "the long job finished") {
		t.Fatalf("the tool result = %q, want the task's final result", got)
	}
	if server.saw("tasks/get") == 0 {
		t.Fatal("the server was never polled: the task path did not run")
	}
	for i, req := range rec.requests() {
		body, _ := json.Marshal(req)
		for _, leak := range []string{taskHandle, "tasks/get", "tasks/update", "tasks/cancel", "resultType"} {
			if strings.Contains(string(body), leak) {
				t.Fatalf("request %d shows the model %q: the task protocol must stay host-side", i, leak)
			}
		}
	}
}
