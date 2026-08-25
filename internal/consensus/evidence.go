package consensus

import (
	"errors"
	"sort"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type EvidenceKind string

const (
	DoubleProposal EvidenceKind = "double_proposal"
	DoublePrevote  EvidenceKind = "double_prevote"
	InvalidCommit  EvidenceKind = "invalid_commit"
	OldVote        EvidenceKind = "old_vote"
)

type Evidence struct {
	ID        string
	Kind      EvidenceKind
	Validator string
	Height    chain_domain.Height
	View      chain_domain.View
	First     chain_domain.Vote
	Second    chain_domain.Vote
	Reason    string
	CreatedAt time.Time
	Resolved  bool
}

type EvidencePool struct {
	mu    sync.RWMutex
	items map[string]Evidence
	max   int
}

func NewEvidencePool(max int) *EvidencePool {
	return &EvidencePool{items: map[string]Evidence{}, max: max}
}
func (p *EvidencePool) Add(e Evidence) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if e.ID == "" || e.Validator == "" {
		return errors.New("evidence identity required")
	}
	if _, ok := p.items[e.ID]; ok {
		return errors.New("evidence duplicate")
	}
	if len(p.items) >= p.max {
		return errors.New("evidence full")
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	p.items[e.ID] = e
	return nil
}
func (p *EvidencePool) Get(id string) (Evidence, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.items[id]
	return e, ok
}
func (p *EvidencePool) List() []Evidence {
	p.mu.RLock()
	defer p.mu.RUnlock()
	o := make([]Evidence, 0, len(p.items))
	for _, e := range p.items {
		o = append(o, e)
	}
	sort.Slice(o, func(i, j int) bool { return o[i].CreatedAt.Before(o[j].CreatedAt) })
	return o
}
func (p *EvidencePool) Resolve(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.items[id]
	if !ok {
		return errors.New("evidence missing")
	}
	e.Resolved = true
	p.items[id] = e
	return nil
}
func (p *EvidencePool) Unresolved() []Evidence {
	out := []Evidence{}
	for _, e := range p.List() {
		if !e.Resolved {
			out = append(out, e)
		}
	}
	return out
}
func (p *EvidencePool) Count() int { p.mu.RLock(); defer p.mu.RUnlock(); return len(p.items) }

type RoundState struct {
	Height    chain_domain.Height
	View      chain_domain.View
	Phase     Phase
	Proposer  string
	Proposals map[string]chain_domain.Block
	Votes     map[string]chain_domain.Vote
	StartedAt time.Time
	Deadline  time.Time
}

func NewRound(h chain_domain.Height, v chain_domain.View, p string) RoundState {
	return RoundState{Height: h, View: v, Phase: Proposal, Proposer: p, Proposals: map[string]chain_domain.Block{}, Votes: map[string]chain_domain.Vote{}, StartedAt: time.Now(), Deadline: time.Now().Add(5 * time.Second)}
}
func (r *RoundState) AddProposal(b chain_domain.Block) error {
	if r.Phase != Proposal {
		return errors.New("proposal phase closed")
	}
	if b.Header.Height != r.Height || b.Header.View != r.View {
		return errors.New("proposal round mismatch")
	}
	if _, ok := r.Proposals[b.Hash]; ok {
		return errors.New("proposal duplicate")
	}
	r.Proposals[b.Hash] = b
	return nil
}
func (r *RoundState) AddVote(v chain_domain.Vote) error {
	if v.Height != r.Height || v.View != r.View {
		return errors.New("vote round mismatch")
	}
	if _, ok := r.Votes[v.Validator]; ok {
		return errors.New("vote duplicate")
	}
	r.Votes[v.Validator] = v
	return nil
}
func (r *RoundState) Advance() Phase {
	switch r.Phase {
	case Proposal:
		r.Phase = Prevote
	case Prevote:
		r.Phase = Precommit
	case Precommit:
		r.Phase = Commit
	case Commit:
		r.View++
		r.Phase = Proposal
	default:
		r.Phase = Proposal
	}
	return r.Phase
}
func (r RoundState) TimedOut(now time.Time) bool { return now.After(r.Deadline) }
func (r RoundState) VoteCount() int              { return len(r.Votes) }
func (r RoundState) ProposalCount() int          { return len(r.Proposals) }
func (r RoundState) Committed() bool             { return r.Phase == Commit }
func (r RoundState) Valid() bool                 { return r.Height > 0 && r.Proposer != "" && !r.StartedAt.IsZero() }

type Schedule struct {
	mu         sync.RWMutex
	validators []chain_domain.Validator
	epoch      chain_domain.Height
	history    map[chain_domain.Height][]chain_domain.Validator
}

func NewSchedule(v []chain_domain.Validator, epoch chain_domain.Height) *Schedule {
	return &Schedule{validators: append([]chain_domain.Validator(nil), v...), epoch: epoch, history: map[chain_domain.Height][]chain_domain.Validator{}}
}
func (s *Schedule) At(h chain_domain.Height) []chain_domain.Validator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	best := chain_domain.Height(0)
	out := s.validators
	for x, v := range s.history {
		if x <= h && x >= best {
			best = x
			out = v
		}
	}
	return append([]chain_domain.Validator(nil), out...)
}
func (s *Schedule) Rotate(h chain_domain.Height, v []chain_domain.Validator) error {
	if h == 0 || len(v) == 0 {
		return errors.New("invalid rotation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[h] = append([]chain_domain.Validator(nil), v...)
	return nil
}
func (s *Schedule) Epoch(h chain_domain.Height) uint64 {
	if s.epoch == 0 {
		return 0
	}
	return uint64(h / s.epoch)
}
func (s *Schedule) Proposer(h chain_domain.Height, v chain_domain.View) string {
	vs := s.At(h)
	if len(vs) == 0 {
		return ""
	}
	return vs[int(v)%len(vs)].Address
}
func (s *Schedule) History() map[chain_domain.Height][]chain_domain.Validator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o := map[chain_domain.Height][]chain_domain.Validator{}
	for h, v := range s.history {
		o[h] = append([]chain_domain.Validator(nil), v...)
	}
	return o
}
func (s *Schedule) Validator(h chain_domain.Height, id string) bool {
	for _, v := range s.At(h) {
		if v.Address == id {
			return true
		}
	}
	return false
}
func (s *Schedule) Power(h chain_domain.Height) uint64 {
	var p uint64
	for _, v := range s.At(h) {
		p += v.Power
	}
	return p
}
func (s *Schedule) Quorum(h chain_domain.Height) uint64 { return s.Power(h)*2/3 + 1 }
