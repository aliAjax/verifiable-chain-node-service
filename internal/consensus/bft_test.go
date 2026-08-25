package consensus

import (
	"testing"
	"time"

	"verifiable-chain-node/internal/chain_domain"
)

func TestFinalizedDoesNotBlock(t *testing.T) {
	e := New([]chain_domain.Validator{{Address: "a", Power: 1}, {Address: "b", Power: 1}, {Address: "c", Power: 1}})
	e.phase = Commit
	e.votes["a"] = chain_domain.Vote{Validator: "a"}
	e.votes["b"] = chain_domain.Vote{Validator: "b"}
	e.votes["c"] = chain_domain.Vote{Validator: "c"}
	done := make(chan bool, 1)
	go func() { done <- e.Finalized() }()
	select {
	case finalized := <-done:
		if !finalized {
			t.Fatal("expected finalized")
		}
	case <-time.After(time.Second):
		t.Fatal("Finalized blocked")
	}
}
