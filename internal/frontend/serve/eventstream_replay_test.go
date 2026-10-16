package serve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
	"reasonix/internal/session/control"
)

// askFor parks an Ask through the controller and returns a cancel that ends it
// once the test is done with the stop.
func askFor(t *testing.T, ctrl *control.Controller, question string) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	askCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = ctrl.Ask(askCtx, []event.AskQuestion{{
			ID: "q1", Prompt: question, Options: []event.AskOption{{Label: "A"}, {Label: "B"}},
		}})
	}()
	return cancel, done
}

// replayFrames is the polled transport's answer, read as the client reads it.
type replayFrames struct {
	Frames []struct {
		Kind      string `json:"kind"`
		Seq       int64  `json:"seq"`
		Watermark int64  `json:"watermark"`
		Ask       struct {
			ID string `json:"id"`
		} `json:"ask"`
	} `json:"frames"`
	Complete  bool  `json:"complete"`
	Watermark int64 `json:"watermark"`
}

func getReplay(t *testing.T, handler http.Handler, query string) replayFrames {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/events/replay"+query, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /events/replay%s = %d, want 200", query, rec.Code)
	}
	var answer replayFrames
	if err := json.Unmarshal(rec.Body.Bytes(), &answer); err != nil {
		t.Fatalf("replay answer is unreadable: %v (%s)", err, rec.Body.String())
	}
	return answer
}

// A page that attaches over the polled transport has to be asked what the run is
// waiting on, exactly as the stream asks it. The prompt is what a run is blocked
// on, so a page that attached without it draws a stopped run with nothing to
// answer — and no later frame brings the question back.
func TestEventsReplayHandsAnAttachingClientTheWaitingAsk(t *testing.T) {
	bc := NewBroadcaster()
	ctrl := control.New(control.Options{Sink: bc})
	ctrl.EnableInteractiveApproval()
	srv := New(ctrl, bc, config.ServeConfig{})
	cancelAsk, asking := askFor(t, ctrl, "pick one")
	defer func() { cancelAsk(); <-asking }()

	handler := operatorHandler(srv)
	deadline := time.Now().Add(3 * time.Second)
	var asked bool
	for !asked {
		for _, f := range getReplay(t, handler, "").Frames {
			if f.Kind == "ask_request" {
				if f.Ask.ID == "" {
					t.Fatal("the attach carried no answerable ask")
				}
				asked = true
				// Connection-local: no number, so a client cannot read it as log
				// history and advance its cursor past frames it never saw.
				if f.Seq != 0 {
					t.Fatalf("the attach handoff is numbered (%d), which would move a client's cursor", f.Seq)
				}
			}
		}
		if !asked {
			if time.Now().After(deadline) {
				t.Fatal("attaching client never received the waiting ask_request")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// An attach is not a resume. A resuming client carries a cursor and reads the
// numbered log, where the prompt also lives — but the attach path must not add a
// second, connection-local copy on top of what the log already holds.
func TestEventsReplayAddsNothingToAResume(t *testing.T) {
	bc := NewBroadcaster()
	ctrl := control.New(control.Options{Sink: bc})
	ctrl.EnableInteractiveApproval()
	srv := New(ctrl, bc, config.ServeConfig{})
	cancelAsk, asking := askFor(t, ctrl, "pick one")
	defer func() { cancelAsk(); <-asking }()

	handler := operatorHandler(srv)
	deadline := time.Now().Add(3 * time.Second)
	for !ctrl.PendingPrompt() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the parked ask")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The log holds the prompt as a numbered frame; a resume replays it as such.
	fromLog := getReplay(t, handler, "?lastEventId=0")
	var logged bool
	for _, f := range fromLog.Frames {
		if f.Kind == "ask_request" {
			logged = f.Seq > 0
		}
	}
	if !logged {
		t.Fatalf("the resume did not replay the numbered prompt from the log: %+v", fromLog.Frames)
	}

	// Nothing is injected ahead of a resume: the frame count matches the log.
	logFrames, _ := bc.Replay(0)
	if got, want := len(fromLog.Frames), len(logFrames); got != want {
		t.Fatalf("resume frames = %d, log frames = %d; the resume carried more than the log", got, want)
	}
}

// Answering is what clears it: a page that attached and answered must not be
// handed the same question again on the next attach.
func TestEventsReplayDropsAnAskOnceItIsAnswered(t *testing.T) {
	bc := NewBroadcaster()
	ctrl := control.New(control.Options{Sink: bc})
	ctrl.EnableInteractiveApproval()
	srv := New(ctrl, bc, config.ServeConfig{})
	cancelAsk, asking := askFor(t, ctrl, "pick one")
	defer func() { cancelAsk(); <-asking }()

	handler := operatorHandler(srv)
	deadline := time.Now().Add(3 * time.Second)
	var askID string
	for askID == "" {
		for _, f := range getReplay(t, handler, "").Frames {
			if f.Kind == "ask_request" {
				askID = f.Ask.ID
			}
		}
		if askID == "" {
			if time.Now().After(deadline) {
				t.Fatal("attaching client never received the waiting ask_request")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if err := ctrl.AnswerQuestion(askID, []event.AskAnswer{{QuestionID: "q1", Selected: []string{"A"}}}); err != nil {
		t.Fatalf("answering the parked ask failed: %v", err)
	}
	<-asking
	cancelAsk()

	for _, f := range getReplay(t, handler, "").Frames {
		if f.Kind == "ask_request" {
			t.Fatalf("a settled ask was replayed to the next attach: %+v", f)
		}
	}
	if strings.Contains(getReplay(t, handler, "?lastEventId=0").Frames[0].Kind, "ask_request") {
		t.Fatal("a settled ask came back through the log")
	}
}
