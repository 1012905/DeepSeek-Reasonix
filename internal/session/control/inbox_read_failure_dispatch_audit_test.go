package control

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/event"
)

// TestDispatchStopsOnFailedQueueRead pins the user-visible half of the
// queue-read contract: an item the dispatcher could not read must still be
// delivered when the queue becomes readable again, without a new user kick.
// The transaction path is broken the production way — transaction.lock is a
// directory — while the manifest and blob stay intact and the queue has work.
func TestDispatchStopsOnFailedQueueRead(t *testing.T) {
	dir := testenv.TempDir(t)
	inputs := make(chan string, 8)
	c := New(Options{
		Runner:      inboxReadFailureRunner{inputs: inputs},
		Sink:        event.FuncSink(func(event.Event) {}),
		SessionDir:  dir,
		SessionPath: filepath.Join(dir, "session.jsonl"),
	})
	t.Cleanup(func() {
		c.Close()
		c.autosaveWG.Wait()
	})

	if _, err := c.EnqueueInbox(InboxRequest{Submit: "must not be lost to a failed scan"}); err != nil {
		t.Fatal(err)
	}
	st, err := c.ensureInbox()
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	scans, scansFound := 0, 0
	retries := make(chan func(), 8)
	c.inbox.mu.Lock()
	c.inbox.afterDispatchScan = func(found bool) {
		mu.Lock()
		defer mu.Unlock()
		scans++
		if found {
			scansFound++
		}
	}
	// Take over the retry timer so the schedule is driven by this test instead
	// of by wall-clock delays.
	c.inbox.scheduleDispatchRetry = func(_ time.Duration, retry func()) {
		retries <- retry
	}
	c.inbox.mu.Unlock()

	lock := filepath.Join(st.Dir(), "transaction.lock")
	if err := os.Remove(lock); err != nil {
		t.Fatalf("remove lock file: %v", err)
	}
	if err := os.Mkdir(lock, 0o700); err != nil {
		t.Fatalf("replace lock with directory: %v", err)
	}

	// One dispatch turn with a broken transaction path. It must either deliver
	// the item or keep retrying; a bounded burst that gives up while the durable
	// item is still queued is the defect.
	returned := make(chan struct{})
	go func() {
		c.maybeDispatchInbox()
		close(returned)
	}()
	select {
	case got := <-inputs:
		t.Fatalf("dispatched %q through a transaction that cannot have run", got)
	case <-returned:
	}

	// Drain the whole retry budget while the transaction path is still broken.
	// Every attempt fails; each one schedules the next.
	for range 4 {
		select {
		case retry := <-retries:
			retry()
		default:
		}
	}
	// Let any synchronous tail of the last retry finish before counting scans.
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		quiet := len(retries) == 0
		mu.Unlock()
		if quiet || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case got := <-inputs:
		t.Fatalf("dispatched %q through a transaction that cannot have run", got)
	default:
	}

	mu.Lock()
	scanCount, foundCount := scans, scansFound
	mu.Unlock()
	// Drain any retry the dispatcher produced but never consumed: exhausting the
	// budget is the point, and a leftover entry proves it gave up mid-burst.
	for {
		select {
		case <-retries:
			continue
		default:
		}
		break
	}
	snap := c.InboxSnapshot()
	t.Logf("dispatcher gave up after %d scans (%d saw the queued item); queue still holds %d item(s)",
		scanCount, foundCount, len(snap.Items))
	if foundCount == 0 {
		t.Fatalf("no scan saw the queued item; this test did not exercise the scan path")
	}
	if len(snap.Items) != 1 {
		t.Fatalf("queued item disappeared: %+v", snap)
	}

	// Restore a working transaction path and give the dispatcher every chance to
	// finish the work it already knows about, without a new user-visible kick.
	if err := os.Remove(lock); err != nil {
		t.Fatalf("remove lock directory: %v", err)
	}
	select {
	case got := <-inputs:
		if got != "must not be lost to a failed scan" {
			t.Fatalf("dispatched input = %q", got)
		}
	case <-time.After(3 * time.Second):
		mu.Lock()
		finalScans := scans
		mu.Unlock()
		t.Fatalf("queued prompt was never dispatched after the failed queue read: "+
			"dispatcher gave up after %d scans while the durable item stayed queued; "+
			"a failed queue read ended dispatch with no retry",
			finalScans)
	}
}

type inboxReadFailureRunner struct {
	inputs chan string
}

func (r inboxReadFailureRunner) Run(_ context.Context, input string) error {
	r.inputs <- input
	return nil
}
