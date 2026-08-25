package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	Requests atomic.Uint64
	Blocks   atomic.Uint64
	Errors   atomic.Uint64
	Started  time.Time

	mu    sync.Mutex
	peers map[string]uint64
}

func New() *Metrics { return &Metrics{Started: time.Now(), peers: map[string]uint64{}} }
func (m *Metrics) RecordPeerResult(peer string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[peer]++
}
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	peers := make(map[string]uint64, len(m.peers))
	for k, v := range m.peers {
		peers[k] = v
	}
	return map[string]interface{}{"requests": m.Requests.Load(), "blocks": m.Blocks.Load(), "errors": m.Errors.Load(), "uptime_seconds": time.Since(m.Started).Seconds(), "peer_results": peers}
}
