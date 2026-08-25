package p2p

import (
	"sync"
	"time"
)

type Peer struct {
	ID, Address string
	Score       int
	LastSeen    time.Time
	Banned      bool
}
type Manager struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

func New() *Manager { return &Manager{peers: map[string]Peer{}} }
func (m *Manager) Add(p Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.LastSeen = time.Now()
	m.peers[p.ID] = p
}
func (m *Manager) List() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o := make([]Peer, 0, len(m.peers))
	for _, p := range m.peers {
		o = append(o, p)
	}
	return o
}
func (m *Manager) Penalize(id string, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.peers[id]
	p.Score -= n
	if p.Score < -100 {
		p.Banned = true
	}
	m.peers[id] = p
}
