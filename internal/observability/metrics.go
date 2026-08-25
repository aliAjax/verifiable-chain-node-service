package observability

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	Requests atomic.Uint64
	Blocks   atomic.Uint64
	Errors   atomic.Uint64
	Started  time.Time
	peers    map[string]uint64
}

func New() *Metrics { return &Metrics{Started: time.Now(), peers: map[string]uint64{}} }
func (m *Metrics) RecordPeerResult(peer string) {
	m.peers[peer]++
}
func (m *Metrics) Snapshot() map[string]interface{} {
	return map[string]interface{}{"requests": m.Requests.Load(), "blocks": m.Blocks.Load(), "errors": m.Errors.Load(), "uptime_seconds": time.Since(m.Started).Seconds(), "peer_results": m.peers}
}
