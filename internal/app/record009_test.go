package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestMiddlewareKeepsRequestCancellation(t *testing.T) {
	n := NewNode(DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil).WithContext(ctx)
	seen := make(chan error, 1)
	n.middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen <- r.Context().Err()
	})).ServeHTTP(httptest.NewRecorder(), req)
	if err := <-seen; err != context.Canceled {
		t.Fatalf("handler context error = %v, want context.Canceled", err)
	}
}

func TestCanceledMineRequestDoesNotCommit(t *testing.T) {
	n := NewNode(DefaultConfig())
	n.opMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/blocks/mine", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		n.Handler().ServeHTTP(rr, req)
		close(done)
	}()
	waitForRequest(t, n)
	cancel()
	n.opMu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("mine handler did not finish after cancellation")
	}
	if got := n.repo.Latest().Header.Height; got != 0 {
		t.Fatalf("canceled mine committed height %d", got)
	}
	if rr.Code != http.StatusRequestTimeout {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestTimeout)
	}
}

func TestCanceledSnapshotRequestDoesNotCommit(t *testing.T) {
	n := NewNode(DefaultConfig())
	n.opMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		n.Handler().ServeHTTP(rr, req)
		close(done)
	}()
	waitForRequest(t, n)
	cancel()
	n.opMu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("snapshot handler did not finish after cancellation")
	}
	if got := len(n.repo.Snapshots()); got != 0 {
		t.Fatalf("canceled request stored %d snapshots", got)
	}
	if rr.Code != http.StatusRequestTimeout {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestTimeout)
	}
}

func TestNodeLifecycleStopsSnapshotCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewNodeWithContext(ctx, DefaultConfig())
	n.opMu.Lock()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots", nil)
	rr := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		n.Handler().ServeHTTP(rr, req)
		close(done)
	}()
	waitForRequest(t, n)
	cancel()
	n.opMu.Unlock()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("snapshot handler ignored node shutdown")
	}
	if got := len(n.repo.Snapshots()); got != 0 {
		t.Fatalf("shutdown stored %d snapshots", got)
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func waitForRequest(t *testing.T, n *Node) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for n.metrics.Requests.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("request never entered middleware")
		}
		runtime.Gosched()
	}
}
