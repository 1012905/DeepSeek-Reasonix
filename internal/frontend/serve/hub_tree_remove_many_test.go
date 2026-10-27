package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reasonix/internal/base/testenv"
	"reasonix/internal/state/sessionstore"
	"reasonix/internal/state/store"
	"testing"
)

func postRemoveSessions(t *testing.T, srv *httptest.Server, paths []string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string][]string{"paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(srv.URL+"/tree/sessions/remove-many", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// A batch request erases every named transcript in one shot, and leaves nothing
// behind for the ones it removed.
func TestRemoveSessionsDeletesEachPath(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()

	paths := []string{
		filepath.Join(SessionDirFor(root), "a.jsonl"),
		filepath.Join(SessionDirFor(root), "b.jsonl"),
		filepath.Join(SessionDirFor(root), "c.jsonl"),
	}
	for _, p := range paths {
		writeSessionAt(t, p)
	}

	resp := postRemoveSessions(t, srv, paths)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := readAllString(resp)
		t.Fatalf("POST /tree/sessions/remove-many = %d, want 204: %s", resp.StatusCode, b)
	}
	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("transcript survived the batch delete: %s", p)
		}
		if _, err := os.Stat(store.SessionEventLog(p)); !os.IsNotExist(err) {
			t.Errorf("event log survived the batch delete: %s", p)
		}
	}
}

// One held transcript must refuse the whole batch: nothing is deleted, not even
// the paths that were free. A partial delete would strand a reader mid-way.
func TestRemoveSessionsRefusesALeasedTranscript(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()

	a := filepath.Join(SessionDirFor(root), "a.jsonl")
	b := filepath.Join(SessionDirFor(root), "b.jsonl")
	writeSessionAt(t, a)
	writeSessionAt(t, b)

	lease, err := sessionstore.TryAcquireSessionLease(a)
	if err != nil {
		t.Fatalf("TryAcquireSessionLease: %v", err)
	}
	defer lease.Release()

	resp := postRemoveSessions(t, srv, []string{a, b})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		b, _ := readAllString(resp)
		t.Fatalf("POST /tree/sessions/remove-many = %d, want 409 while one path is held: %s", resp.StatusCode, b)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("a refused batch deleted %s anyway: %v", p, err)
		}
	}
}

// A path that is not inside a known workspace is rejected, and the valid paths
// in the same request are left untouched.
func TestRemoveSessionsRefusesAnUnknownWorkspace(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()

	inside := filepath.Join(SessionDirFor(root), "inside.jsonl")
	writeSessionAt(t, inside)
	outside := filepath.Join(testenv.TempDir(t), "outside.jsonl")
	writeSessionAt(t, outside)

	resp := postRemoveSessions(t, srv, []string{inside, outside})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		b, _ := readAllString(resp)
		t.Fatalf("POST /tree/sessions/remove-many = %d, want 403 for an out-of-workspace path: %s", resp.StatusCode, b)
	}
	if _, err := os.Stat(inside); err != nil {
		t.Errorf("a refused batch deleted the in-workspace session: %v", err)
	}
}

// An empty list is not a valid request; the endpoint answers before touching
// anything on disk.
func TestRemoveSessionsRefusesAnEmptyList(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()

	resp := postRemoveSessions(t, srv, nil)
	defer resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Fatalf("POST /tree/sessions/remove-many = %d, want a client error for an empty list", resp.StatusCode)
	}
}

// Duplicates in the request are erased once, not repeatedly — the second mention
// must not turn into a second guard acquisition against an already-gone file.
func TestRemoveSessionsDeduplicatesPaths(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	root := testenv.TempDir(t)
	h := NewHub(HubOptions{})
	hubRuntime(t, h, root)
	srv := httptest.NewServer(operatorHandler(h))
	defer srv.Close()

	p := filepath.Join(SessionDirFor(root), "dup.jsonl")
	writeSessionAt(t, p)

	resp := postRemoveSessions(t, srv, []string{p, p})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := readAllString(resp)
		t.Fatalf("POST /tree/sessions/remove-many = %d, want 204 for a deduplicated list: %s", resp.StatusCode, b)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("transcript survived the delete: %v", err)
	}
}
