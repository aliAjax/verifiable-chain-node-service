package repository

import (
	"errors"
	"sort"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type Index struct {
	mu        sync.RWMutex
	heights   []chain_domain.Height
	hashes    map[string]chain_domain.Height
	proposers map[string][]chain_domain.Height
	created   time.Time
}

func NewIndex() *Index {
	return &Index{hashes: map[string]chain_domain.Height{}, proposers: map[string][]chain_domain.Height{}, created: time.Now()}
}

func (i *Index) Add(b chain_domain.Block) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if b.Hash == "" {
		return errors.New("hash required")
	}
	if _, ok := i.hashes[b.Hash]; ok {
		return errors.New("hash already indexed")
	}
	for _, height := range i.heights {
		if height == b.Header.Height {
			return errors.New("height already indexed")
		}
	}
	i.heights = append(i.heights, b.Header.Height)
	i.hashes[b.Hash] = b.Header.Height
	i.proposers[b.Header.Proposer] = append(i.proposers[b.Header.Proposer], b.Header.Height)
	return nil
}

func (i *Index) Height(hash string) (chain_domain.Height, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	h, ok := i.hashes[hash]
	return h, ok
}

func (i *Index) Proposer(name string) []chain_domain.Height {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]chain_domain.Height(nil), i.proposers[name]...)
}

func (i *Index) Heights() []chain_domain.Height {
	i.mu.RLock()
	defer i.mu.RUnlock()
	o := append([]chain_domain.Height(nil), i.heights...)
	sort.Slice(o, func(a, b int) bool { return o[a] < o[b] })
	return o
}

func (i *Index) Count() int         { i.mu.RLock(); defer i.mu.RUnlock(); return len(i.heights) }
func (i *Index) Empty() bool        { return i.Count() == 0 }
func (i *Index) Created() time.Time { i.mu.RLock(); defer i.mu.RUnlock(); return i.created }

func (i *Index) RemoveAbove(h chain_domain.Height) {
	i.mu.Lock()
	defer i.mu.Unlock()
	newHeights := make([]chain_domain.Height, 0, len(i.heights))
	for _, x := range i.heights {
		if x <= h {
			newHeights = append(newHeights, x)
		}
	}
	i.heights = newHeights
	for hash, x := range i.hashes {
		if x > h {
			delete(i.hashes, hash)
		}
	}
	for p, xs := range i.proposers {
		keep := xs[:0]
		for _, x := range xs {
			if x <= h {
				keep = append(keep, x)
			}
		}
		if len(keep) == 0 {
			delete(i.proposers, p)
		} else {
			i.proposers[p] = keep
		}
	}
}

type BlockMeta struct {
	Height    chain_domain.Height
	Hash      string
	Size      int
	TxCount   int
	StateRoot string
	CreatedAt time.Time
}
type MetaStore struct {
	mu       sync.RWMutex
	byHeight map[chain_domain.Height]BlockMeta
	byHash   map[string]BlockMeta
}

func NewMetaStore() *MetaStore {
	return &MetaStore{byHeight: map[chain_domain.Height]BlockMeta{}, byHash: map[string]BlockMeta{}}
}
func (s *MetaStore) Put(m BlockMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.byHeight[m.Height]; ok && old.Hash != m.Hash {
		delete(s.byHash, old.Hash)
	}
	if old, ok := s.byHash[m.Hash]; ok && old.Height != m.Height {
		delete(s.byHeight, old.Height)
	}
	s.byHeight[m.Height] = m
	s.byHash[m.Hash] = m
}
func (s *MetaStore) Get(h chain_domain.Height) (BlockMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.byHeight[h]
	return m, ok
}
func (s *MetaStore) GetHash(hash string) (BlockMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.byHash[hash]
	return m, ok
}
func (s *MetaStore) DeleteAbove(h chain_domain.Height) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for x, m := range s.byHeight {
		if x > h {
			delete(s.byHeight, x)
			delete(s.byHash, m.Hash)
		}
	}
}
func (s *MetaStore) All() []BlockMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := make([]BlockMeta, 0, len(s.byHeight))
	for _, m := range s.byHeight {
		o = append(o, m)
	}
	sort.Slice(o, func(a, b int) bool { return o[a].Height < o[b].Height })
	return o
}

type Query struct {
	From     chain_domain.Height
	To       chain_domain.Height
	Proposer string
	Limit    int
}

func (s *MetaStore) Query(q Query) []BlockMeta {
	o := s.All()
	out := []BlockMeta{}
	for _, m := range o {
		if q.From > 0 && m.Height < q.From {
			continue
		}
		if q.To > 0 && m.Height > q.To {
			continue
		}
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
		out = append(out, m)
	}
	return out
}
func (s *MetaStore) Size() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.byHeight) }
func (s *MetaStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byHeight = map[chain_domain.Height]BlockMeta{}
	s.byHash = map[string]BlockMeta{}
}
