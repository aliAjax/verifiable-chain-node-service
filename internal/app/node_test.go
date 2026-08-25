package app

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestConcurrentMineCommitsOneHeightAtATime(t *testing.T) {
	n := NewNode(DefaultConfig())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest(http.MethodPost, "/api/v1/blocks/mine", nil)
			w := httptest.NewRecorder()
			n.Handler().ServeHTTP(w, r)
			if w.Code != http.StatusCreated {
				t.Errorf("mine status = %d", w.Code)
			}
		}()
	}
	wg.Wait()
	if got := n.repo.Latest().Header.Height; got != 8 {
		t.Fatalf("height = %d, want 8", got)
	}
}
