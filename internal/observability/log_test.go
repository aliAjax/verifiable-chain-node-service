package observability

import "testing"

func TestCounterSnapshotIsDetached(t *testing.T) {
	c := NewCounter("requests")
	c.Label("network", "alpha")
	s := c.Snapshot()
	labels := s["labels"].(map[string]string)
	labels["network"] = "changed"
	again := c.Snapshot()["labels"].(map[string]string)
	if again["network"] != "alpha" {
		t.Fatal("snapshot labels mutated counter")
	}
}
