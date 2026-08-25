package proof

import "testing"

func TestMerkleProofRoundTrip(t *testing.T) {
	items := map[string]string{"alpha": "1", "beta": "2", "gamma": "3"}
	for key := range items {
		p, err := Proof(items, key)
		if err != nil {
			t.Fatalf("proof %s: %v", key, err)
		}
		if !VerifyProof(p) {
			t.Fatalf("proof %s did not verify", key)
		}
		p.Value += "x"
		if VerifyProof(p) {
			t.Fatalf("tampered proof %s verified", key)
		}
	}
}

func TestMissingMerkleKey(t *testing.T) {
	if _, err := Proof(map[string]string{"alpha": "1"}, "missing"); err == nil {
		t.Fatal("missing key returned a proof")
	}
}
