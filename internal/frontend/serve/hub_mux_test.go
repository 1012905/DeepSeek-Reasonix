package serve

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/event"
)

type muxFrame struct {
	Kind  string `json:"kind"`
	Conn  string `json:"conn"`
	Pane  string `json:"pane"`
	Frame struct {
		Kind string `json:"kind"`
		Seq  int64  `json:"seq"`
		Text string `json:"text"`
		Code string `json:"code"`
	} `json:"frame"`
}

type muxClient struct {
	t      *testing.T
	srv    *httptest.Server
	conn   string
	cancel context.CancelFunc
	frames chan muxFrame
}

func openMux(t *testing.T, srv *httptest.Server) *muxClient {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events/mux", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	c := &muxClient{t: t, srv: srv, cancel: cancel, frames: make(chan muxFrame, 256)}
	t.Cleanup(cancel)
	go func() {
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var f muxFrame
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &f) == nil {
				c.frames <- f
			}
		}
	}()
	open := c.next()
	if open.Kind != "mux_open" || len(open.Conn) != 32 {
		t.Fatalf("first frame = %+v, want mux_open with a 128-bit conn id", open)
	}
	c.conn = open.Conn
	return c
}

func (c *muxClient) next() muxFrame {
	c.t.Helper()
	select {
	case f := <-c.frames:
		return f
	case <-time.After(3 * time.Second):
		c.t.Fatal("no frame within 3s")
		return muxFrame{}
	}
}

func (c *muxClient) call(method, pane, body string) (int, string) {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.srv.URL+"/events/mux/"+c.conn+"/panes/"+pane, strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func notice(code string) event.Event {
	return event.Event{Kind: event.Notice, Code: code, Detail: code}
}

func muxHub(t *testing.T, panes int) (*Hub, []*Runtime, *httptest.Server) {
	t.Helper()
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	h := NewHub(HubOptions{})
	var rts []*Runtime
	for range panes {
		rts = append(rts, hubRuntime(t, h, testenv.TempDir(t)))
	}
	srv := httptest.NewServer(operatorHandler(h))
	t.Cleanup(srv.Close)
	return h, rts, srv
}

// One stream carries every pane, each frame in an envelope naming its pane, and
// a frame never reaches a pane that did not produce it.
func TestMuxCarriesEachPaneInItsOwnEnvelope(t *testing.T) {
	_, rts, srv := muxHub(t, 2)
	c := openMux(t, srv)
	for _, rt := range rts {
		if code, body := c.call(http.MethodPut, rt.ID, ""); code != http.StatusNoContent {
			t.Fatalf("attach %s = %d %s", rt.ID, code, body)
		}
	}
	rts[0].Events.Emit(notice("a-only"))
	rts[1].Events.Emit(notice("b-only"))
	seen := map[string]string{}
	for len(seen) < 2 {
		f := c.next()
		if f.Frame.Code != "" && f.Frame.Seq > 0 {
			seen[f.Pane] = f.Frame.Code
		}
	}
	if seen[rts[0].ID] != "a-only" || seen[rts[1].ID] != "b-only" {
		t.Fatalf("envelopes = %v, want each pane to carry only its own frame", seen)
	}
}

// Subscribing again with a cursor resumes from it: the frames after it arrive
// once each, and the old subscription does not keep feeding the connection.
func TestMuxResubscribeResumesFromTheCursorWithoutDuplicates(t *testing.T) {
	_, rts, srv := muxHub(t, 1)
	rt := rts[0]
	c := openMux(t, srv)
	c.call(http.MethodPut, rt.ID, "")
	rt.Events.Emit(notice("one"))
	rt.Events.Emit(notice("two"))
	var last int64
	for got := 0; got < 2; {
		if f := c.next(); f.Frame.Seq > 0 && f.Frame.Code != "" {
			got++
		}
	}
	rt.Events.Emit(notice("three"))
	for f := c.next(); ; f = c.next() {
		if f.Frame.Code == "three" {
			last = f.Frame.Seq
			break
		}
	}
	if code, _ := c.call(http.MethodPut, rt.ID, `{"after":`+jsonInt(last)+`}`); code != http.StatusNoContent {
		t.Fatalf("resubscribe = %d", code)
	}
	rt.Events.Emit(notice("four"))
	var codes []string
	deadline := time.After(2 * time.Second)
collect:
	for len(codes) < 1 {
		select {
		case f := <-c.frames:
			if f.Frame.Seq > last && f.Frame.Code != "" {
				codes = append(codes, f.Frame.Code)
			}
		case <-deadline:
			break collect
		}
	}
	select {
	case f := <-c.frames:
		if f.Frame.Code != "" && f.Frame.Seq > 0 {
			t.Fatalf("extra frame %+v after the resume: the old subscription is still feeding", f)
		}
	case <-time.After(200 * time.Millisecond):
	}
	if strings.Join(codes, ",") != "four" {
		t.Fatalf("after cursor %d got %v, want [four] once", last, codes)
	}
}

func jsonInt(n int64) string { b, _ := json.Marshal(n); return string(b) }

// A connection that goes away takes every subscription with it.
func TestMuxDisconnectReleasesEverySubscription(t *testing.T) {
	_, rts, srv := muxHub(t, 3)
	c := openMux(t, srv)
	base := make([]int, len(rts))
	for i, rt := range rts {
		base[i] = rt.Events.Subscribers()
		c.call(http.MethodPut, rt.ID, "")
		if got := rt.Events.Subscribers(); got != base[i]+1 {
			t.Fatalf("%s subscribers = %d, want %d", rt.ID, got, base[i]+1)
		}
	}
	c.cancel()
	deadline := time.Now().Add(3 * time.Second)
	for {
		left := 0
		for i, rt := range rts {
			left += rt.Events.Subscribers() - base[i]
		}
		if left == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d subscriptions outlived the connection", left)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if code, body := c.call(http.MethodPut, rts[0].ID, ""); code != http.StatusNotFound || !strings.Contains(body, codeMuxUnknownConn) {
		t.Fatalf("attach on a closed stream = %d %s, want 404 %s", code, body, codeMuxUnknownConn)
	}
}

func TestMuxRefusalsCarryTypedCodes(t *testing.T) {
	_, rts, srv := muxHub(t, 1)
	c := openMux(t, srv)
	if code, body := c.call(http.MethodPut, "nope", ""); code != http.StatusNotFound || !strings.Contains(body, codeNotFound) {
		t.Fatalf("unknown pane = %d %s", code, body)
	}
	c.conn = strings.Repeat("0", 32)
	if code, body := c.call(http.MethodPut, rts[0].ID, ""); code != http.StatusNotFound || !strings.Contains(body, codeMuxUnknownConn) {
		t.Fatalf("unknown conn = %d %s", code, body)
	}
}

func TestMuxTellsTheClientWhenAPaneCloses(t *testing.T) {
	h, rts, srv := muxHub(t, 2)
	c := openMux(t, srv)
	for _, rt := range rts {
		c.call(http.MethodPut, rt.ID, "")
	}
	if err := h.Close(rts[0].ID); err != nil {
		t.Fatal(err)
	}
	for {
		f := c.next()
		if f.Frame.Kind == "mux_pane_closed" {
			if f.Pane != rts[0].ID {
				t.Fatalf("closed pane = %s, want %s", f.Pane, rts[0].ID)
			}
			return
		}
	}
}

// The writer takes one frame from each pane in turn, so a pane with a backlog
// cannot keep a quiet one waiting.
func TestMuxNextRotatesBetweenPanes(t *testing.T) {
	c := &muxConn{panes: map[string]*muxPane{}}
	for _, id := range []string{"busy", "quiet"} {
		c.panes[id] = &muxPane{id: id, slot: make(chan Frame, 1)}
		c.order = append(c.order, id)
	}
	cursor := 0
	var order []string
	for i := range 4 {
		c.panes["busy"].slot <- Frame{Seq: 1, Data: []byte(`{}`)}
		if i == 1 {
			c.panes["quiet"].slot <- Frame{Seq: 1, Data: []byte(`{}`)}
		}
		data, ok := c.next(&cursor)
		if !ok {
			t.Fatal("no frame")
		}
		var f muxFrame
		_ = json.Unmarshal(data, &f)
		order = append(order, f.Pane)
		for len(c.panes["busy"].slot) > 0 {
			<-c.panes["busy"].slot
		}
	}
	if strings.Join(order, ",") != "busy,quiet,busy,busy" {
		t.Fatalf("order = %v, want the quiet pane served as soon as it had a frame", order)
	}
}

// A dropped connection costs nothing: the client opens a new stream, names each
// pane's last seq, and every pane's remaining frames arrive once, in order.
func TestMuxReconnectLosesAndRepeatsNoFramePerPane(t *testing.T) {
	_, rts, srv := muxHub(t, 3)
	const total = 60
	emitted := map[string][]string{}
	got := map[string][]string{}
	cursor := map[string]int64{}
	record := func(f muxFrame) {
		if f.Frame.Seq <= cursor[f.Pane] || f.Frame.Code == "" {
			return
		}
		cursor[f.Pane] = f.Frame.Seq
		got[f.Pane] = append(got[f.Pane], f.Frame.Code)
	}
	attach := func(c *muxClient) {
		for _, rt := range rts {
			body := ""
			if cursor[rt.ID] > 0 {
				body = `{"after":` + jsonInt(cursor[rt.ID]) + `}`
			}
			if code, b := c.call(http.MethodPut, rt.ID, body); code != http.StatusNoContent {
				t.Fatalf("attach %s = %d %s", rt.ID, code, b)
			}
		}
	}
	emit := func(from, to int) {
		for i := from; i < to; i++ {
			for _, rt := range rts {
				code := rt.ID + "-" + jsonInt(int64(i))
				emitted[rt.ID] = append(emitted[rt.ID], code)
				rt.Events.Emit(notice(code))
			}
		}
	}
	drain := func(c *muxClient, quiet time.Duration) {
		for {
			select {
			case f := <-c.frames:
				record(f)
			case <-time.After(quiet):
				return
			}
		}
	}

	first := openMux(t, srv)
	attach(first)
	emit(0, 20)
	drain(first, 300*time.Millisecond)
	first.cancel()
	emit(20, 40)

	second := openMux(t, srv)
	attach(second)
	emit(40, total)
	drain(second, 500*time.Millisecond)

	for _, rt := range rts {
		if strings.Join(got[rt.ID], ",") != strings.Join(emitted[rt.ID], ",") {
			t.Fatalf("%s: got %d frames, want %d, in order and once each\n got: %v\nwant: %v", rt.ID, len(got[rt.ID]), len(emitted[rt.ID]), got[rt.ID], emitted[rt.ID])
		}
	}
}
