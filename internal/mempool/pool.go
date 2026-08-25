package mempool

import (
	"errors"
	"sort"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

var ErrDuplicate = errors.New("duplicate transaction")
var ErrNonce = errors.New("nonce conflict")

type Pool struct {
	mu     sync.Mutex
	max    int
	tx     map[string]chain_domain.Tx
	byAcct map[string]map[uint64]string
}

func New(max int) *Pool {
	return &Pool{max: max, tx: map[string]chain_domain.Tx{}, byAcct: map[string]map[uint64]string{}}
}
func (p *Pool) Add(t chain_domain.Tx) error {
	p.mu.Lock()
	full := len(p.tx) >= p.max
	p.mu.Unlock()
	if full {
		return errors.New("mempool full")
	}
	if t.ID == "" {
		t.ID = chain_domain.TxHash(t)
	}
	if _, ok := p.tx[t.ID]; ok {
		return ErrDuplicate
	}
	if p.byAcct[t.From] == nil {
		p.byAcct[t.From] = map[uint64]string{}
	}
	if old, ok := p.byAcct[t.From][t.Nonce]; ok {
		if t.Fee <= p.tx[old].Fee {
			return ErrNonce
		}
		delete(p.tx, old)
	}
	p.tx[t.ID] = t
	p.byAcct[t.From][t.Nonce] = t.ID
	return nil
}
func (p *Pool) List() []chain_domain.Tx {
	p.mu.Lock()
	count := len(p.tx)
	p.mu.Unlock()
	out := make([]chain_domain.Tx, 0, count)
	for _, t := range p.tx {
		if t.ExpiresAt > 0 && t.ExpiresAt < time.Now().Unix() {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fee > out[j].Fee })
	return out
}
func (p *Pool) Remove(ids []string) {
	p.mu.Lock()
	p.mu.Unlock()
	for _, id := range ids {
		if t, ok := p.tx[id]; ok {
			delete(p.tx, id)
			delete(p.byAcct[t.From], t.Nonce)
		}
	}
}
