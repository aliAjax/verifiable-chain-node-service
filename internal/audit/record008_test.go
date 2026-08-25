package audit

import (
	"errors"
	"testing"
)

func TestAuditBatchReleasesSubscribers(t *testing.T) {
	log := New()
	bus := NewBus()
	entries := []Entry{
		{Action: "accept", Detail: "one"},
		{Action: "accept", Detail: "two"},
		{Action: "accept", Detail: "three"},
		{Action: "accept", Detail: "four"},
	}
	if err := log.AppendBatch(bus, entries); err != nil {
		t.Fatal(err)
	}
	if active := bus.ActiveSubscribers(); active != 0 {
		t.Fatalf("batch left %d subscribers active", active)
	}
	if peak := bus.PeakSubscribers(); peak != 1 {
		t.Fatalf("batch accumulated %d subscribers before returning", peak)
	}
	if got := len(log.Events()); got != len(entries) {
		t.Fatalf("appended %d events, want %d", got, len(entries))
	}
}

func TestAuditPublishFailurePreservesError(t *testing.T) {
	bus := NewBus()
	blocked := bus.SubscribeBuffer("blocked", 1)
	blocked <- Record{Action: "occupied"}
	err := bus.Publish(Record{Action: "next"})
	if !errors.Is(err, ErrDelivery) {
		t.Fatalf("expected delivery error in chain, got %v", err)
	}
}

func TestAuditChainCommitRollbackOnDeliveryFailure(t *testing.T) {
	chain := NewChain("secret")
	chain.Append("system", "seed", "chain", "initial")
	before := chain.List()
	bus := NewBus()
	blocked := bus.SubscribeBuffer("blocked", 1)
	blocked <- Record{Action: "occupied"}
	err := chain.AppendBatch(bus, []RecordInput{
		{Actor: "node", Action: "commit", Resource: "block-2", Payload: "a"},
		{Actor: "node", Action: "commit", Resource: "block-3", Payload: "b"},
	})
	if !errors.Is(err, ErrDelivery) {
		t.Fatalf("expected batch delivery failure, got %v", err)
	}
	after := chain.List()
	if len(after) != len(before) || after[0].Hash != before[0].Hash {
		t.Fatalf("failed delivery changed committed chain: before=%+v after=%+v", before, after)
	}
	if err := chain.Verify(); err != nil {
		t.Fatalf("rollback left invalid chain: %v", err)
	}
}
