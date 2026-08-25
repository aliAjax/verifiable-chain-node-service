package p2p

import (
	"sync"
	"testing"
	"time"
)

func TestPeerConcurrentPenaltyBanIsolation(t *testing.T) {
	manager := New()
	manager.Add(Peer{ID: "peer-a", Score: 100})
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(4)
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 300; i++ {
			manager.Penalize("peer-a", 1)
		}
	}()
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 300; i++ {
			manager.Ban("peer-a", time.Second)
			manager.Unban("peer-a")
		}
	}()
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 300; i++ {
			_ = manager.List()
			_ = manager.HealthyCount()
		}
	}()
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 300; i++ {
			manager.Add(Peer{ID: "peer-b", Score: i})
		}
	}()
	close(start)
	wait.Wait()
	peers := manager.List()
	if len(peers) != 2 {
		t.Fatalf("peer snapshot lost an entry: %+v", peers)
	}
	var target Peer
	for _, peer := range peers {
		if peer.ID == "peer-a" {
			target = peer
		}
	}
	if target.Score != -200 {
		t.Fatalf("penalties lost during concurrent updates: score=%d", target.Score)
	}
}

func TestPeerBackoffResetDoesNotLoseState(t *testing.T) {
	backoff := NewBackoff()
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(3)
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 500; i++ {
			backoff.Next("peer-a")
		}
	}()
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 500; i++ {
			backoff.Reset("peer-a")
		}
	}()
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 500; i++ {
			backoff.Next("peer-b")
		}
	}()
	close(start)
	wait.Wait()
	if got := backoff.Next("peer-b"); got != 256*time.Second {
		t.Fatalf("independent peer backoff was lost: %s", got)
	}
}
