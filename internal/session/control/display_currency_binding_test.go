package control

import (
	"os"
	"testing"

	"reasonix/internal/base/testenv"
	"reasonix/internal/contract/config"
	"reasonix/internal/contract/event"
)

func TestDisplayCurrencyBindingIsQuietCanonicalAndFailAtomic(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(config.UserConfigPath(), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("[billing]\ndisplay_currency = \"USD\"\n")
	var group DisplayCurrencyGroup
	oldContext, newContext := &event.QuoteContext{}, &event.QuoteContext{}
	notices := 0
	sink := event.FuncSink(func(event.Event) { notices++ })
	old := New(Options{Sink: sink, QuoteContext: oldContext})
	next := New(Options{Sink: sink, QuoteContext: newContext})
	t.Cleanup(old.Close)
	t.Cleanup(next.Close)
	display := ""
	project := func(currency string) { display = currency }
	if err := group.Bind(nil, old, project); err != nil {
		t.Fatal(err)
	}
	write("[billing]\ndisplay_currency = [")
	if err := group.Bind(old, next, project); err == nil {
		t.Fatal("malformed canonical configuration accepted")
	}
	if len(group.members) != 1 || next.currencyGroup() != nil || display != "USD" {
		t.Fatal("failed binding changed membership or projection")
	}
	write("[billing]\ndisplay_currency = \"CNY\"\n")
	if err := group.Bind(old, next, project); err != nil {
		t.Fatal(err)
	}
	if newContext.DisplayCurrency != "CNY" || display != "CNY" || notices != 0 {
		t.Fatalf("canonical quiet projection = %q %q, notices %d", newContext.DisplayCurrency, display, notices)
	}
	if err := group.Bind(next, next, project); err != nil {
		t.Fatal(err)
	}
	if notices != 0 || len(group.members) != 1 {
		t.Fatal("same-generation bind duplicated notice or membership")
	}
	old.Close()
	if len(group.members) != 1 {
		t.Fatal("old teardown removed replacement")
	}
	if err := group.Bind(old, next, project); err == nil {
		t.Fatal("retired previous controller re-enrolled")
	}
	next.Close()
	if err := group.Bind(nil, next, project); err == nil {
		t.Fatal("closed controller re-enrolled")
	}
	if len(group.members) != 0 {
		t.Fatal("closed member remained registered")
	}
}

func TestDisplayCurrencyBindingOrdersProjectionBeforeFanout(t *testing.T) {
	t.Setenv("REASONIX_HOME", testenv.TempDir(t))
	if err := os.WriteFile(config.UserConfigPath(), []byte("[billing]\ndisplay_currency = \"CNY\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var group DisplayCurrencyGroup
	quote := &event.QuoteContext{}
	display := ""
	sink := event.NewCostQuoteSink(event.FuncSink(func(e event.Event) {
		if e.Code == event.NoticeCodeDisplayCurrency {
			display = e.Detail
		}
	}), quote)
	next := New(Options{Sink: sink, QuoteContext: quote})
	t.Cleanup(next.Close)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- group.Bind(nil, next, func(currency string) { close(entered); <-release; display = currency })
	}()
	<-entered
	if group.mu.TryLock() {
		group.mu.Unlock()
		t.Error("projection callback runs outside group serialization")
	}
	emitted := make(chan struct{})
	go func() {
		group.mu.Lock()
		group.emitLocked(next, event.Event{Kind: event.Notice, Code: event.NoticeCodeDisplayCurrency, Detail: "USD"})
		group.mu.Unlock()
		close(emitted)
	}()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	<-emitted
	if quote.DisplayCurrency != "USD" || display != "USD" {
		t.Fatalf("projection overtook notice: quote %q, frontend %q", quote.DisplayCurrency, display)
	}
}
