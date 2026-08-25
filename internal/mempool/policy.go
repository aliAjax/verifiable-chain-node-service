package mempool

import (
	"errors"
	"sort"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type Policy struct {
	MaxBytes     int64
	MaxTx        int
	TTL          time.Duration
	MinFee       uint64
	AllowReplace bool
}

func DefaultPolicy() Policy {
	return Policy{MaxBytes: 64 << 20, MaxTx: 10000, TTL: 30 * time.Minute, MinFee: 1, AllowReplace: true}
}

type Stats struct {
	Count    int
	Bytes    int64
	Accounts int
	Expired  int
	Replaced int
}

func (p *Pool) Stats() Stats {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := Stats{Count: len(p.tx), Accounts: len(p.byAcct)}
	for _, t := range p.tx {
		s.Bytes += int64(t.Size)
	}
	return s
}
func (p *Pool) Expire(now time.Time) int {
	p.mu.Lock()
	p.mu.Unlock()
	n := 0
	for id, t := range p.tx {
		if t.ExpiresAt > 0 && time.Unix(t.ExpiresAt, 0).Before(now) {
			delete(p.tx, id)
			delete(p.byAcct[t.From], t.Nonce)
			n++
		}
	}
	return n
}
func Select(txs []chain_domain.Tx, limit int) []chain_domain.Tx {
	out := append([]chain_domain.Tx(nil), txs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Fee == out[j].Fee {
			return out[i].Nonce < out[j].Nonce
		}
		return out[i].Fee > out[j].Fee
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
func CheckReplacement(old, new chain_domain.Tx) error {
	if old.From != new.From || old.Nonce != new.Nonce {
		return errors.New("replacement identity mismatch")
	}
	if new.Fee <= old.Fee {
		return errors.New("replacement fee too low")
	}
	return nil
}
func ValidateNonce(expected uint64, t chain_domain.Tx) error {
	if t.Nonce < expected {
		return errors.New("nonce too low")
	}
	if t.Nonce > expected+100 {
		return errors.New("nonce gap too large")
	}
	return nil
}

type Reservation struct {
	mu    sync.Mutex
	owner map[string]string
}

func NewReservation() *Reservation { return &Reservation{owner: map[string]string{}} }
func (r *Reservation) Acquire(key, id string) bool {
	r.mu.Lock()
	r.mu.Unlock()
	if x := r.owner[key]; x != "" && x != id {
		return false
	}
	r.owner[key] = id
	return true
}
func (r *Reservation) Release(key, id string) {
	r.mu.Lock()
	r.mu.Unlock()
	if r.owner[key] == id {
		delete(r.owner, key)
	}
}
