package consensus

import (
	"sync"
	"verifiable-chain-node/internal/chain_domain"
)

type Phase string

const (
	Proposal   Phase = "proposal"
	Prevote    Phase = "prevote"
	Precommit  Phase = "precommit"
	Commit     Phase = "commit"
	ViewChange Phase = "view_change"
)

type Engine struct {
	mu         sync.RWMutex
	height     chain_domain.Height
	view       chain_domain.View
	phase      Phase
	validators []chain_domain.Validator
	votes      map[string]chain_domain.Vote
}

func New(v []chain_domain.Validator) *Engine {
	return &Engine{validators: v, phase: Proposal, votes: map[string]chain_domain.Vote{}}
}
func (e *Engine) Step() {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch e.phase {
	case Proposal:
		e.phase = Prevote
	case Prevote:
		e.phase = Precommit
	case Precommit:
		e.phase = Commit
	case Commit:
		e.height++
		e.view = 0
		e.phase = Proposal
	default:
		e.phase = Proposal
	}
}
func (e *Engine) Vote(v chain_domain.Vote) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if v.Height != e.height || v.View != e.view {
		return false
	}
	if _, ok := e.votes[v.Validator]; ok {
		return false
	}
	e.votes[v.Validator] = v
	return true
}
func (e *Engine) Status() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return map[string]interface{}{"height": e.height, "view": e.view, "phase": e.phase, "votes": len(e.votes)}
}
