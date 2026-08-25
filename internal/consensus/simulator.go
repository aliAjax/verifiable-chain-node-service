package consensus

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type Message struct {
	From      string
	To        string
	Height    chain_domain.Height
	View      chain_domain.View
	Phase     Phase
	BlockHash string
	SentAt    time.Time
}
type Network struct {
	mu      sync.Mutex
	queues  map[string][]Message
	dropped map[string]int
	latency time.Duration
}

func NewNetwork() *Network {
	return &Network{queues: map[string][]Message{}, dropped: map[string]int{}, latency: time.Millisecond}
}
func (n *Network) Send(m Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if m.To == "" || m.From == "" {
		return errors.New("route required")
	}
	n.queues[m.To] = append(n.queues[m.To], m)
	return nil
}
func (n *Network) Recv(id string) []Message {
	n.mu.Lock()
	defer n.mu.Unlock()
	o := n.queues[id]
	delete(n.queues, id)
	return o
}
func (n *Network) DropPeer(id string)    { n.mu.Lock(); defer n.mu.Unlock(); n.dropped[id]++ }
func (n *Network) Dropped(id string) int { n.mu.Lock(); defer n.mu.Unlock(); return n.dropped[id] }
func (e *Engine) SetHeight(h chain_domain.Height) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.height = h
	e.votes = map[string]chain_domain.Vote{}
}
func (e *Engine) SetView(v chain_domain.View) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.view = v
	e.phase = Proposal
	e.votes = map[string]chain_domain.Vote{}
}
func (e *Engine) Quorum() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return chain_domain.Quorum(e.validators)
}
func (e *Engine) Finalized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.phase == Commit && uint64(len(e.votes)) >= chain_domain.Quorum(e.validators)
}
func (e *Engine) Proposer() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.validators) == 0 {
		return ""
	}
	return e.validators[int(e.view)%len(e.validators)].Address
}
func ValidateVote(v chain_domain.Vote, phase Phase) error {
	if v.Validator == "" || v.Height == 0 {
		return errors.New("vote identity missing")
	}
	if v.Type != string(phase) {
		return fmt.Errorf("phase mismatch: %s", v.Type)
	}
	if v.Signature == "" {
		return errors.New("signature missing")
	}
	return nil
}
func SimulateRound(e *Engine, net *Network, b chain_domain.Block) []Message {
	out := []Message{}
	for _, v := range e.validators {
		for _, p := range []Phase{Proposal, Prevote, Precommit} {
			m := Message{From: v.Address, To: v.Address, Height: b.Header.Height, View: b.Header.View, Phase: p, BlockHash: b.Hash, SentAt: time.Now()}
			_ = net.Send(m)
			out = append(out, m)
		}
	}
	return out
}
