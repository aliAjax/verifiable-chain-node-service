package repository

import (
	"testing"
	"verifiable-chain-node/internal/chain_domain"
)

func TestRepoRejectsDuplicateBlockHash(t *testing.T) {
	r := New()
	if err := r.PutBlock(chain_domain.Block{Hash: "same", Header: chain_domain.Header{Height: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := r.PutBlock(chain_domain.Block{Hash: "same", Header: chain_domain.Header{Height: 2}}); err == nil {
		t.Fatal("same block hash accepted at a second height")
	}
}

func TestIndexRejectsDuplicateHeight(t *testing.T) {
	i := NewIndex()
	if err := i.Add(chain_domain.Block{Hash: "h1", Header: chain_domain.Header{Height: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := i.Add(chain_domain.Block{Hash: "h2", Header: chain_domain.Header{Height: 1}}); err == nil {
		t.Fatal("second hash accepted at an indexed height")
	}
}

func TestMetaStoreReplacementRemovesOldHash(t *testing.T) {
	s := NewMetaStore()
	s.Put(BlockMeta{Height: 1, Hash: "old"})
	s.Put(BlockMeta{Height: 1, Hash: "new"})
	if _, ok := s.GetHash("old"); ok {
		t.Fatal("old hash still resolves after height replacement")
	}
	if m, ok := s.GetHash("new"); !ok || m.Height != 1 {
		t.Fatalf("new hash missing: ok=%v meta=%+v", ok, m)
	}
}

func TestMetaStoreHashMoveRemovesOldHeight(t *testing.T) {
	s := NewMetaStore()
	s.Put(BlockMeta{Height: 1, Hash: "same"})
	s.Put(BlockMeta{Height: 2, Hash: "same"})
	if _, ok := s.Get(1); ok {
		t.Fatal("old height remains after hash moved")
	}
	if m, ok := s.GetHash("same"); !ok || m.Height != 2 {
		t.Fatalf("hash did not move: ok=%v meta=%+v", ok, m)
	}
}

func TestIndexRemoveAbovePrunesProposerBuckets(t *testing.T) {
	i := NewIndex()
	if err := i.Add(chain_domain.Block{Hash: "h1", Header: chain_domain.Header{Height: 1, Proposer: "old"}}); err != nil {
		t.Fatal(err)
	}
	if err := i.Add(chain_domain.Block{Hash: "h2", Header: chain_domain.Header{Height: 2, Proposer: "new"}}); err != nil {
		t.Fatal(err)
	}
	i.RemoveAbove(1)
	if _, ok := i.proposers["new"]; ok {
		t.Fatal("future proposer bucket survived rollback")
	}
	if got := i.Proposer("old"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("retained proposer index = %v", got)
	}
}
