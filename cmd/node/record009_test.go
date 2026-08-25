package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNodeFactoryKeepsShutdownContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := newNode(ctx)
	cancel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/snapshots", nil)
	n.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d after shutdown", rr.Code, http.StatusServiceUnavailable)
	}
}
