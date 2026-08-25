package observability

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type overlapWriter struct {
	active  atomic.Int32
	overlap atomic.Bool
	gate    chan struct{}
	once    sync.Once
}

func newOverlapWriter() *overlapWriter { return &overlapWriter{gate: make(chan struct{})} }

func (w *overlapWriter) Write(p []byte) (int, error) {
	if w.active.Add(1) == 2 {
		w.overlap.Store(true)
		w.once.Do(func() { close(w.gate) })
	}
	select {
	case <-w.gate:
	case <-time.After(30 * time.Millisecond):
	}
	w.active.Add(-1)
	return len(p), nil
}

func TestDerivedLoggersSerializeWriter(t *testing.T) {
	w := newOverlapWriter()
	base := NewLogger(w, LevelInfo)
	a := base.With("peer", "a")
	b := base.With("peer", "b")
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, logger := range []*Logger{a, b} {
		wg.Add(1)
		go func(l *Logger) {
			defer wg.Done()
			<-start
			l.Info(context.Background(), "connected", nil)
		}(logger)
	}
	close(start)
	wg.Wait()
	if w.overlap.Load() {
		t.Fatal("derived loggers wrote the shared writer concurrently")
	}
}

func TestCounterSnapshotLabelsDetached(t *testing.T) {
	c := NewCounter("blocks")
	c.Label("network", "alpha")
	snapshot := c.Snapshot()
	labels := snapshot["labels"].(map[string]string)
	labels["network"] = "mutated"
	again := c.Snapshot()["labels"].(map[string]string)
	if got := again["network"]; got != "alpha" {
		t.Fatalf("stored label = %q, want alpha", got)
	}
}

func TestMetricsPeerResultConcurrent(t *testing.T) {
	m := New()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				m.RecordPeerResult("peer-a")
				_ = m.Snapshot()
			}
		}()
	}
	close(start)
	wg.Wait()
	peers := m.Snapshot()["peer_results"].(map[string]uint64)
	if got := peers["peer-a"]; got != 1200 {
		t.Fatalf("peer result count = %d, want 1200", got)
	}
}

func TestMetricsSnapshotDetached(t *testing.T) {
	m := New()
	m.RecordPeerResult("peer-a")
	peers := m.Snapshot()["peer_results"].(map[string]uint64)
	peers["peer-a"] = 99
	again := m.Snapshot()["peer_results"].(map[string]uint64)
	if got := again["peer-a"]; got != 1 {
		t.Fatalf("stored peer result = %d, want 1", got)
	}
}
