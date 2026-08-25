package repository

import (
	"errors"
	"sync"
	"verifiable-chain-node/internal/chain_domain"
)

type Repo struct {
	mu     sync.RWMutex
	blocks map[chain_domain.Height]chain_domain.Block
	snaps  map[string]chain_domain.Snapshot
}

func New() *Repo {
	return &Repo{blocks: map[chain_domain.Height]chain_domain.Block{}, snaps: map[string]chain_domain.Snapshot{}}
}
func (r *Repo) PutBlock(b chain_domain.Block) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.blocks[b.Header.Height]; ok {
		return errors.New("block exists")
	}
	for _, existing := range r.blocks {
		if existing.Hash != "" && existing.Hash == b.Hash {
			return errors.New("block hash exists")
		}
	}
	r.blocks[b.Header.Height] = cloneBlock(b)
	return nil
}
func (r *Repo) Block(h chain_domain.Height) (chain_domain.Block, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.blocks[h]
	return cloneBlock(b), ok
}
func (r *Repo) Latest() chain_domain.Block {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var b chain_domain.Block
	for _, x := range r.blocks {
		if x.Header.Height > b.Header.Height {
			b = x
		}
	}
	return cloneBlock(b)
}
func (r *Repo) PutSnapshot(s chain_domain.Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snaps[s.ID] = cloneSnapshot(s)
}
func (r *Repo) Snapshot(id string) (chain_domain.Snapshot, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.snaps[id]
	return cloneSnapshot(s), ok
}
func (r *Repo) Snapshots() []chain_domain.Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := make([]chain_domain.Snapshot, 0, len(r.snaps))
	for _, s := range r.snaps {
		o = append(o, cloneSnapshot(s))
	}
	return o
}

func cloneBlock(b chain_domain.Block) chain_domain.Block {
	b.Transactions = append([]chain_domain.Tx(nil), b.Transactions...)
	b.Commit = append([]chain_domain.Vote(nil), b.Commit...)
	return b
}

func cloneSnapshot(s chain_domain.Snapshot) chain_domain.Snapshot {
	data := make(map[string]string, len(s.Data))
	for k, v := range s.Data {
		data[k] = v
	}
	s.Data = data
	return s
}
