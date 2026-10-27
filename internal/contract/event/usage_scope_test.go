package event

import (
	"encoding/json"
	"testing"

	"reasonix/internal/contract/provider"
)

type usageScopeTestSink struct {
	AuditForwarder
	current any
	events  []Event
}

func (s *usageScopeTestSink) CaptureUsageScope() any { return s.current }
func (s *usageScopeTestSink) Emit(e Event)           { s.events = append(s.events, e) }

func TestUsageScopePreservesCapturedOwnerAndWire(t *testing.T) {
	old, next := make(chan struct{}), make(chan struct{})
	sink := &usageScopeTestSink{current: old}
	chain := Coalesce(Sync(NewCostQuoteSink(sink, nil)), DefaultStreamDeltaWindow)
	bound := BindUsageScope(chain)
	sink.current = next
	nested := BindUsageScope(bound)
	u := Event{Kind: Usage, Usage: &provider.Usage{PromptTokens: 10}, UsageSource: UsageSourceSubagent}
	nested.Emit(u)
	if len(sink.events) != 1 || UsageScope(sink.events[0]) != old {
		t.Fatal("nested capture adopted a later owner")
	}
	chain.Emit(u)
	want, err := json.Marshal(sink.events[1])
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(sink.events[0])
	if err != nil || string(got) != string(want) {
		t.Fatalf("scope changed JSON: %s %v", got, err)
	}
	if UsageScope(u) != nil {
		t.Fatal("capture mutated the producer event")
	}
	var replay Event
	if err := json.Unmarshal(got, &replay); err != nil || UsageScope(replay) != nil {
		t.Fatal("replay acquired process-local ownership")
	}
	bound.Emit(Event{Kind: Notice, Text: "unchanged"})
	if len(sink.events) != 3 || UsageScope(sink.events[2]) != nil {
		t.Fatal("non-usage event acquired attribution")
	}
}

func TestUsageScopeKeepsEarlierEventOwner(t *testing.T) {
	sink := &usageScopeTestSink{current: "new"}
	BindUsageScope(sink).Emit(Event{Kind: Usage, usageScope: "old"})
	if UsageScope(sink.events[0]) != "old" {
		t.Fatal("wrapper overwrote earlier attribution")
	}
}

func TestUsageScopeWithoutCapability(t *testing.T) {
	if BindUsageScope(nil) != nil {
		t.Fatal("nil sink changed")
	}
	var typedNil *usageScopeTestSink
	if BindUsageScope(typedNil) != typedNil {
		t.Fatal("typed nil sink changed")
	}
	var calls int
	sink := FuncSink(func(e Event) {
		calls++
		if UsageScope(e) != nil {
			t.Fatal("unscoped sink gained attribution")
		}
	})
	BindUsageScope(sink).Emit(Event{Kind: Usage})
	if calls != 1 {
		t.Fatal("unscoped event dropped")
	}
}
