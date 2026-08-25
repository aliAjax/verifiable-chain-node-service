package repository

import (
	"testing"

	"verifiable-chain-node/internal/chain_domain"
)

func TestRepositoryReturnsDetachedValues(t *testing.T) {
	r := New()
	b := chain_domain.Block{Header: chain_domain.Header{Height: 1}, Hash: "block-1", Transactions: []chain_domain.Tx{{ID: "tx-1"}}}
	if err := r.PutBlock(b); err != nil {
		t.Fatal(err)
	}
	got, ok := r.Block(1)
	if !ok {
		t.Fatal("block missing")
	}
	got.Transactions[0].ID = "changed"
	again, _ := r.Block(1)
	if again.Transactions[0].ID != "tx-1" {
		t.Fatal("block transaction escaped repository")
	}

	r.PutSnapshot(chain_domain.Snapshot{ID: "snap", Data: map[string]string{"a": "1"}})
	s, _ := r.Snapshot("snap")
	s.Data["a"] = "2"
	againSnapshot, _ := r.Snapshot("snap")
	if againSnapshot.Data["a"] != "1" {
		t.Fatal("snapshot data escaped repository")
	}
}
