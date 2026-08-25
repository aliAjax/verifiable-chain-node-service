package repository

import (
	"errors"
	"sort"
	"verifiable-chain-node/internal/chain_domain"
)

func (r *Repo) Range(from, to chain_domain.Height) []chain_domain.Block {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o := []chain_domain.Block{}
	for h := from; h <= to; h++ {
		if b, ok := r.blocks[h]; ok {
			o = append(o, cloneBlock(b))
		}
	}
	sort.Slice(o, func(i, j int) bool { return o[i].Header.Height < o[j].Header.Height })
	return o
}
func (r *Repo) VerifyChain() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var prev *chain_domain.Block
	for h := chain_domain.Height(1); h <= chain_domain.Height(len(r.blocks)); h++ {
		b, ok := r.blocks[h]
		if !ok {
			return errors.New("chain gap")
		}
		if e := chain_domain.ValidateBlock(b, prev); e != nil {
			return e
		}
		cp := b
		prev = &cp
	}
	return nil
}
func (r *Repo) DeleteAbove(h chain_domain.Height) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for x := range r.blocks {
		if x > h {
			delete(r.blocks, x)
		}
	}
}
