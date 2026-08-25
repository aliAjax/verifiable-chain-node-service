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
}

func New() *Metrics { return &Metrics{Started: time.Now()} }
func (m *Metrics) Snapshot() map[string]interface{} {
	return map[string]interface{}{"requests": m.Requests.Load(), "blocks": m.Blocks.Load(), "errors": m.Errors.Load(), "uptime_seconds": time.Since(m.Started).Seconds()}
}
