package consensus

import (
	"sync"
	"testing"

	"verifiable-chain-node/internal/chain_domain"
)

func validators004() []chain_domain.Validator {
	return []chain_domain.Validator{{Address: "a", Power: 1}, {Address: "b", Power: 1}, {Address: "c", Power: 1}}
}

func TestViewChangeRejectsLateVotes(t *testing.T) {
	e := New(validators004())
	e.SetHeight(1)
	if !e.Vote(chain_domain.Vote{Validator: "a", Height: 1, View: 0}) {
		t.Fatal("initial vote rejected")
	}
	start := make(chan struct{})
	changed := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		e.SetView(1)
		close(changed)
	}()
	go func() {
		defer wg.Done()
		<-start
		<-changed
		if e.Vote(chain_domain.Vote{Validator: "b", Height: 1, View: 0}) {
			t.Error("late vote accepted")
		}
	}()
	close(start)
	wg.Wait()
	if got := e.Status()["votes"].(int); got != 0 {
		t.Fatalf("votes after view change = %d", got)
	}
}

func TestCommitStartsWithEmptyVoteSet(t *testing.T) {
	e := New(validators004())
	e.SetHeight(1)
	_ = e.Vote(chain_domain.Vote{Validator: "a", Height: 1})
	start := make(chan struct{})
	stepped := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 4; i++ {
			e.Step()
		}
		close(stepped)
	}()
	go func() {
		defer wg.Done()
		<-start
		<-stepped
		_ = e.Status()
	}()
	close(start)
	wg.Wait()
	if got := e.Status()["votes"].(int); got != 0 {
		t.Fatalf("votes after commit = %d", got)
	}
}

func TestRoundDeliveryCompletesWithoutReplay(t *testing.T) {
	network := NewNetwork()
	if err := network.Send(Message{From: "a", To: "b", Height: 1}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	counts := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			counts <- len(network.Recv("b"))
		}()
	}
	close(start)
	wg.Wait()
	close(counts)
	total := 0
	for count := range counts {
		total += count
	}
	if total != 1 {
		t.Fatalf("delivered %d messages, want 1", total)
	}
}

func TestRoundAdvanceResetsNewView(t *testing.T) {
	round := NewRound(1, 0, "a")
	b := chain_domain.Block{Header: chain_domain.Header{Height: 1, View: 0}, Hash: "block"}
	if err := round.AddProposal(b); err != nil {
		t.Fatal(err)
	}
	if err := round.AddVote(chain_domain.Vote{Validator: "a", Height: 1, View: 0}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 4; i++ {
			round.Advance()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 64; i++ {
			_ = round.VoteCount()
			_ = round.ProposalCount()
		}
	}()
	close(start)
	wg.Wait()
	if round.View != 1 || round.Phase != Proposal || round.VoteCount() != 0 || round.ProposalCount() != 0 {
		t.Fatalf("new round not reset: view=%d phase=%s votes=%d proposals=%d", round.View, round.Phase, round.VoteCount(), round.ProposalCount())
	}
}
