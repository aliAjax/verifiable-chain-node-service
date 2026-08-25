package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutPropagatesDeadline(t *testing.T) {
	seen := make(chan time.Time, 1)
	h := Timeout(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			seen <- time.Time{}
			return
		}
		seen <- deadline
	}), 50*time.Millisecond)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/sync", nil))
	deadline := <-seen
	if deadline.IsZero() || time.Until(deadline) > 100*time.Millisecond {
		t.Fatalf("deadline was not propagated: %v", deadline)
	}
}
