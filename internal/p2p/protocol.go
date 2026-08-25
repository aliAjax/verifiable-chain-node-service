package p2p

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Handshake struct {
	NodeID    string
	NetworkID string
	ChainID   string
	Version   uint32
	Features  []string
	Nonce     uint64
}
type Limits struct {
	MaxPeers      int
	MaxMessage    int
	RatePerSecond int
	Burst         int
}

func DefaultLimits() Limits {
	return Limits{MaxPeers: 64, MaxMessage: 2 << 20, RatePerSecond: 100, Burst: 200}
}
func ValidateHandshake(h Handshake) error {
	if h.NodeID == "" || h.NetworkID == "" || h.ChainID == "" {
		return errors.New("identity missing")
	}
	if h.Version == 0 {
		return errors.New("version missing")
	}
	if len(h.Features) > 32 {
		return errors.New("too many features")
	}
	return nil
}
func Compatible(a, b Handshake) bool {
	return a.NetworkID == b.NetworkID && a.ChainID == b.ChainID && a.Version == b.Version
}

type Backoff struct {
	mu       sync.Mutex
	attempts map[string]int
}

func NewBackoff() *Backoff { return &Backoff{attempts: map[string]int{}} }
func (b *Backoff) Next(id string) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.attempts[id]++
	n := b.attempts[id]
	if n > 8 {
		n = 8
	}
	return time.Duration(1<<n) * time.Second
}
func (b *Backoff) Reset(id string) { b.mu.Lock(); defer b.mu.Unlock(); delete(b.attempts, id) }
func ParseAddr(s string) (string, int, error) {
	h, p, e := net.SplitHostPort(s)
	if e != nil || h == "" {
		return "", 0, errors.New("invalid address")
	}
	var n int
	_, e = fmt.Sscanf(p, "%d", &n)
	if e != nil || n < 1 || n > 65535 {
		return "", 0, errors.New("invalid port")
	}
	return h, n, nil
}
func EncodeTopic(network, topic string) string {
	return strings.Trim(network, "/") + "/" + strings.Trim(topic, "/")
}
func TopicAllowed(topic string) bool {
	for _, p := range []string{"blocks", "transactions", "votes", "snapshots"} {
		if topic == p {
			return true
		}
	}
	return false
}
func (m *Manager) HealthyCount() int {
	n := 0
	for _, p := range m.List() {
		if !p.Banned && p.Score >= 0 {
			n++
		}
	}
	return n
}
func (m *Manager) Ban(id string, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.peers[id]
	p.Banned = true
	p.LastSeen = time.Now().Add(d)
	m.peers[id] = p
}
func (m *Manager) Unban(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.peers[id]
	p.Banned = false
	m.peers[id] = p
}
