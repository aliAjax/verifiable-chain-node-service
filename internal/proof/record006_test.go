package proof

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"verifiable-chain-node/internal/chain_domain"
	"verifiable-chain-node/internal/state_machine"
)

func proofItems(prefix string, count int) map[string]string {
	items := make(map[string]string, count)
	for i := 0; i < count; i++ {
		items[fmt.Sprintf("%s-key-%02d", prefix, i)] = fmt.Sprintf("%s-value-%02d", prefix, i)
	}
	return items
}

func proofState(t *testing.T, prefix string, count int) *state_machine.State {
	t.Helper()
	s := state_machine.New()
	for key, value := range proofItems(prefix, count) {
		if err := s.Apply(chain_domain.Tx{Key: key, Value: value, Gas: 1}); err != nil {
			t.Fatalf("apply %s: %v", key, err)
		}
	}
	return s
}

func runParallel(t *testing.T, workers int, fn func()) {
	t.Helper()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for n := 0; n < 100; n++ {
				fn()
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestBuildTreeLargeInputDoesNotPanic(t *testing.T) {
	items := proofItems("large", 17)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if root := BuildTree(items); root == nil || root.Hash == "" {
				t.Error("large tree did not produce a root")
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestConcurrentBuildTreeRootsRemainIsolated(t *testing.T) {
	left := proofItems("left", 7)
	right := proofItems("right", 13)
	wantLeft := BuildTree(left).Hash
	wantRight := BuildTree(right).Hash
	runParallel(t, 8, func() {
		if got := BuildTree(left).Hash; got != wantLeft {
			t.Errorf("left root changed: got %s want %s", got, wantLeft)
		}
		if got := BuildTree(right).Hash; got != wantRight {
			t.Errorf("right root changed: got %s want %s", got, wantRight)
		}
	})
}

func TestConcurrentMerkleProofsKeepOwnSteps(t *testing.T) {
	left := proofItems("alpha", 16)
	right := proofItems("omega", 23)
	leftKey := "alpha-key-07"
	rightKey := "omega-key-11"
	runParallel(t, 8, func() {
		p, err := Proof(left, leftKey)
		if err != nil || !VerifyProof(p) {
			t.Errorf("left proof invalid: err=%v proof=%+v", err, p)
		}
		p, err = Proof(right, rightKey)
		if err != nil || !VerifyProof(p) {
			t.Errorf("right proof invalid: err=%v proof=%+v", err, p)
		}
	})
}

func TestReturnedMerkleProofStepsDoNotChange(t *testing.T) {
	first, err := Proof(proofItems("first", 16), "first-key-05")
	if err != nil {
		t.Fatal(err)
	}
	want := append([]Step(nil), first.Steps...)
	if _, err := Proof(proofItems("second", 16), "second-key-09"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Steps, want) {
		t.Fatalf("returned steps changed after another proof: got %+v want %+v", first.Steps, want)
	}
}

func TestConcurrentStateProofsKeepOwnPaths(t *testing.T) {
	left := proofState(t, "state-left", 19)
	right := proofState(t, "state-right", 29)
	leftKey := "state-left-key-06"
	rightKey := "state-right-key-14"
	wantLeft := append([]string(nil), Build(left, leftKey).Path...)
	wantRight := append([]string(nil), Build(right, rightKey).Path...)
	runParallel(t, 8, func() {
		if got := Build(left, leftKey).Path; !reflect.DeepEqual(got, wantLeft) {
			t.Errorf("left path changed: got %v want %v", got, wantLeft)
		}
		if got := Build(right, rightKey).Path; !reflect.DeepEqual(got, wantRight) {
			t.Errorf("right path changed: got %v want %v", got, wantRight)
		}
	})
}

func TestReturnedStateProofPathDoesNotChange(t *testing.T) {
	firstState := proofState(t, "saved", 18)
	first := Build(firstState, "saved-key-04")
	want := append([]string(nil), first.Path...)
	secondState := proofState(t, "later", 18)
	_ = Build(secondState, "later-key-12")
	if !reflect.DeepEqual(first.Path, want) {
		t.Fatalf("returned path changed after another build: got %v want %v", first.Path, want)
	}
}
