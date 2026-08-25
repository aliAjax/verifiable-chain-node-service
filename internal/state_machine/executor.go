package state_machine

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type Result struct {
	Applied  int
	Failed   int
	GasUsed  uint64
	Root     string
	Duration time.Duration
	Errors   []string
}
type Executor struct {
	State  *State
	MaxGas uint64
	mu     sync.Mutex
}

func NewExecutor(s *State) *Executor { return &Executor{State: s, MaxGas: 10000000} }
func (x *Executor) Execute(txs []chain_domain.Tx) Result {
	x.mu.Lock()
	defer x.mu.Unlock()
	start := time.Now()
	r := Result{}
	for _, t := range txs {
		if t.Gas > x.MaxGas-r.GasUsed {
			r.Failed++
			r.Errors = append(r.Errors, "gas limit")
			continue
		}
		if e := chain_domain.ValidateTx(t); e != nil {
			r.Failed++
			r.Errors = append(r.Errors, e.Error())
			continue
		}
		if e := x.State.Apply(t); e != nil {
			r.Failed++
			r.Errors = append(r.Errors, e.Error())
			continue
		}
		r.Applied++
		r.GasUsed += t.Gas
	}
	r.Root = x.State.Root()
	r.Duration = time.Since(start)
	return r
}
func (x *Executor) Estimate(t chain_domain.Tx) uint64 {
	if t.Gas == 0 {
		return 21000
	}
	return t.Gas
}
func (x *Executor) ValidateBatch(txs []chain_domain.Tx) error {
	if len(txs) > 4096 {
		return errors.New("batch too large")
	}
	seen := map[string]bool{}
	for _, t := range txs {
		if seen[t.ID] {
			return errors.New("duplicate tx")
		}
		seen[t.ID] = true
	}
	return nil
}
func (x *Executor) ApplyWithRollback(h chain_domain.Height, txs []chain_domain.Tx) (Result, error) {
	x.State.Snapshot(h)
	r := x.Execute(txs)
	if r.Failed > 0 {
		err := x.State.Rollback(h)
		// The root captured during Execute reflects the partially-applied
		// state before rollback. Recompute it so callers see the rolled-back
		// state root, matching what the interface reports after a failure.
		r.Root = x.State.Root()
		return r, err
	}
	return r, nil
}
func (r Result) String() string {
	return fmt.Sprintf("applied=%d failed=%d gas=%d root=%s duration=%s", r.Applied, r.Failed, r.GasUsed, r.Root, r.Duration)
}
func (x *Executor) Health() map[string]interface{} {
	return map[string]interface{}{"max_gas": x.MaxGas, "state_root": x.State.Root(), "checked_at": time.Now()}
}
