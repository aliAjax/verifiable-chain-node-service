package mempool

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"verifiable-chain-node/internal/chain_domain"
)

func txFor(id int) chain_domain.Tx {
	return chain_domain.Tx{ID: fmt.Sprintf("tx-%d", id), From: fmt.Sprintf("account-%d", id%8), Nonce: uint64(id), Fee: uint64(id + 1), ExpiresAt: time.Now().Add(time.Hour).Unix()}
}

func TestMempoolConcurrentAdds(t *testing.T) {
	p := New(4096)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			if err := p.Add(txFor(id)); err != nil {
				t.Errorf("add %d: %v", id, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if got := p.Stats().Count; got != 128 {
		t.Fatalf("count = %d, want 128", got)
	}
}

func TestMempoolConcurrentSnapshotIsolation(t *testing.T) {
	p := New(8192)
	for i := 0; i < 256; i++ {
		if err := p.Add(txFor(i)); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			if id%2 == 0 {
				_ = p.List()
				return
			}
			_ = p.Add(txFor(1000 + id))
		}(i)
	}
	close(start)
	wg.Wait()
}

func TestMempoolConcurrentExpiryReplacement(t *testing.T) {
	p := New(8192)
	now := time.Now()
	for i := 0; i < 256; i++ {
		tx := txFor(i)
		tx.ExpiresAt = now.Add(-time.Second).Unix()
		if err := p.Add(tx); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			if id%2 == 0 {
				p.Expire(now)
				return
			}
			_ = p.Add(txFor(2000 + id))
		}(i)
	}
	close(start)
	wg.Wait()
}

func TestReservationConcurrentOwnership(t *testing.T) {
	r := NewReservation()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			owner := fmt.Sprintf("worker-%d", id)
			if r.Acquire("shared", owner) {
				r.Release("shared", owner)
			}
		}(i)
	}
	close(start)
	wg.Wait()
}
