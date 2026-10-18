package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/session/control"
)

func schedulesServer(t *testing.T) (string, string) {
	t.Helper()
	s := newProviderEditServer(t)
	s.AllowProviderEdit()
	srv := httptest.NewServer(operatorHandler(s))
	t.Cleanup(srv.Close)
	return srv.URL, testenv.TempDir(t)
}

func scheduleDraft(ws string) map[string]any {
	return map[string]any{"label": "morning", "everySec": 7200, "workspace": ws, "prompt": "summarize the TODOs"}
}

func TestAScheduleIsPreviewedConfirmedListedAndPaused(t *testing.T) {
	base, ws := schedulesServer(t)
	if got := readJSON[control.ScheduleBoard](t, base, "/schedules"); len(got.Schedules) != 0 || got.PausedAll {
		t.Fatalf("empty board = %+v", got)
	}
	status, raw := postSide(t, base+"/schedules/preview", scheduleDraft(ws))
	if status != http.StatusOK || !strings.Contains(string(raw), `"confirm"`) {
		t.Fatalf("preview = %d %s", status, raw)
	}
	var prev control.SchedulePreview
	mustDecode(t, raw, &prev)

	if status, raw := postSide(t, base+"/schedules", scheduleDraft(ws)); status != http.StatusConflict || !strings.Contains(string(raw), "schedule.confirmation_required") {
		t.Fatalf("no confirmation = %d %s", status, raw)
	}
	changed := scheduleDraft(ws)
	changed["prompt"] = "something else"
	changed["confirm"] = prev.Confirm
	if status, raw := postSide(t, base+"/schedules", changed); status != http.StatusConflict || !strings.Contains(string(raw), "schedule.digest_mismatch") {
		t.Fatalf("a changed draft = %d %s", status, raw)
	}
	ok := scheduleDraft(ws)
	ok["confirm"] = prev.Confirm
	status, raw = postSide(t, base+"/schedules", ok)
	if status != http.StatusOK {
		t.Fatalf("create = %d %s", status, raw)
	}
	var sc control.ScheduleView
	mustDecode(t, raw, &sc)
	if got := readJSON[control.ScheduleBoard](t, base, "/schedules"); len(got.Schedules) != 1 || got.Schedules[0].ID != sc.ID {
		t.Fatalf("board = %+v", got)
	}
	if status, raw := postSide(t, base+"/schedules/"+sc.ID+"/pause", map[string]any{}); status != http.StatusOK || !strings.Contains(string(raw), `"paused"`) {
		t.Fatalf("pause = %d %s", status, raw)
	}
	if status, _ := postSide(t, base+"/schedules/"+sc.ID+"/resume", map[string]any{}); status != http.StatusOK {
		t.Fatalf("resume = %d", status)
	}
	if status, _ := postSide(t, base+"/schedules/pause-all", map[string]any{}); status != http.StatusOK {
		t.Fatalf("pause-all = %d", status)
	}
	if got := readJSON[control.ScheduleBoard](t, base, "/schedules"); !got.PausedAll {
		t.Fatal("the global switch did not show")
	}
	if status, _ := postSide(t, base+"/schedules/"+sc.ID+"/delete", map[string]any{}); status != http.StatusOK {
		t.Fatalf("delete = %d", status)
	}
}

func TestScheduleRefusalsCarryTheirOwnCodes(t *testing.T) {
	base, ws := schedulesServer(t)
	short := scheduleDraft(ws)
	short["everySec"] = 60
	_, raw := postSide(t, base+"/schedules/preview", short)
	var prev control.SchedulePreview
	mustDecode(t, raw, &prev)
	short["confirm"] = prev.Confirm
	for name, c := range map[string]struct {
		path string
		body map[string]any
		code string
	}{
		"under the floor": {"/schedules", short, "schedule.interval_too_short"},
		"no trigger":      {"/schedules/preview", map[string]any{"workspace": ws, "prompt": "x"}, "schedule.bad_trigger"},
		"no folder":       {"/schedules/preview", map[string]any{"everySec": 7200, "workspace": ws + "/nope", "prompt": "x"}, "schedule.no_folder"},
		"unknown model":   {"/schedules/preview", map[string]any{"everySec": 7200, "workspace": ws, "prompt": "x", "model": "nobody/none"}, "schedule.model_unconfigured"},
		"unknown id":      {"/schedules/sch_none/pause", map[string]any{}, "schedule.not_found"},
	} {
		if status, raw := postSide(t, base+c.path, c.body); status < 400 || !strings.Contains(string(raw), c.code) {
			t.Errorf("%s = %d %s, want %s", name, status, raw, c.code)
		}
	}
	resp, err := http.Get(base + "/schedules/result?trigger=sch_x/1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("an unrecorded result = %d", resp.StatusCode)
	}
}

// What widens unattended spending is the machine owner's: a client without the
// edit grant, or a paired phone, can pause and delete but not create or resume.
func TestOnlyTheMachinesOwnerMayCreateOrResumeSchedules(t *testing.T) {
	s := newProviderEditServer(t)
	srv := httptest.NewServer(operatorHandler(s))
	defer srv.Close()
	for _, p := range []string{"/schedules", "/schedules/sch_x/resume", "/schedules/sch_x/renew", "/schedules/resume-all"} {
		if status, raw := postSide(t, srv.URL+p, map[string]any{"confirm": "x"}); status != http.StatusForbidden || !strings.Contains(string(raw), "schedule.editing_disabled") {
			t.Errorf("%s without the grant = %d %s", p, status, raw)
		}
	}
	for _, p := range []string{"/schedules/sch_x/pause", "/schedules/pause-all"} {
		if status, _ := postSide(t, srv.URL+p, map[string]any{}); status == http.StatusForbidden {
			t.Errorf("%s was refused, but only narrows", p)
		}
	}
	s.AllowProviderEdit()
	paired := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		operatorHandler(s).ServeHTTP(w, r.WithContext(withDeviceReach(r.Context(), "phone", 1)))
	}))
	defer paired.Close()
	if status, _ := postSide(t, paired.URL+"/schedules", map[string]any{"confirm": "x", "everySec": 7200, "prompt": "x"}); status == http.StatusOK {
		t.Fatal("a paired device created a schedule")
	}
}

func mustDecode(t *testing.T, raw []byte, into any) {
	t.Helper()
	if err := json.Unmarshal(raw, into); err != nil {
		t.Fatalf("%s: %v", raw, err)
	}
}
