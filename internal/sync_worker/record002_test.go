package sync_worker

import (
	"context"
	"strings"
	"testing"

	"verifiable-chain-node/internal/repository"
)

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestSyncCancellationStopsWork(t *testing.T) {
	w := New(repository.New())
	r := w.Start(cancelledContext(), Request{From: 1, To: 8, Peer: "peer-a"})
	if r.Status != StatusFailed || !strings.Contains(r.Error, context.Canceled.Error()) {
		t.Fatalf("result = %+v", r)
	}
	if r.Downloaded != 0 {
		t.Fatalf("downloaded = %d after cancellation", r.Downloaded)
	}
}

func TestSyncActiveReleasedAfterCancellation(t *testing.T) {
	w := New(repository.New())
	_ = w.Start(cancelledContext(), Request{From: 1, To: 3, Peer: "peer-a"})
	r := w.Start(context.Background(), Request{From: 1, To: 1, Peer: "peer-b"})
	if r.Error == "sync already running" {
		t.Fatal("cancelled run kept worker active")
	}
}

func TestSyncLastPublishesCancelledResult(t *testing.T) {
	w := New(repository.New())
	r := w.Start(cancelledContext(), Request{From: 1, To: 3, Peer: "peer-a"})
	last := w.Last()
	if last.Status != r.Status || last.Error != r.Error || last.FinishedAt.IsZero() {
		t.Fatalf("last = %+v, result = %+v", last, r)
	}
}
