package audit

import "testing"

func TestChainSequenceAndDigest(t *testing.T) {
	c := NewChain("secret")
	for i := 0; i < 4; i++ {
		r := c.Append("node", "commit", "block", "payload")
		if r.Sequence != uint64(i+1) {
			t.Fatalf("sequence = %d, want %d", r.Sequence, i+1)
		}
	}
	if err := c.Verify(); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestChainDetectsTampering(t *testing.T) {
	c := NewChain("secret")
	c.Append("node", "commit", "block", "payload")
	c.records[0].Payload = "changed"
	if err := c.Verify(); err == nil {
		t.Fatal("tampered chain verified")
	}
}
