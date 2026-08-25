package state_machine

import (
	"testing"
	"verifiable-chain-node/internal/chain_domain"
)

func tx(id, key, value string, gas uint64) chain_domain.Tx {
	return chain_domain.Tx{ID: id, From: "a", To: "b", Nonce: 1, Gas: gas, Key: key, Value: value}
}

func TestSnapshotMutationCannotCorruptRollback(t *testing.T) {
	s := New()
	_ = s.Apply(tx("one", "key", "original", 1))
	snap := s.Snapshot(1)
	snap["key"] = "corrupt"
	_ = s.Apply(tx("two", "key", "later", 1))
	if err := s.Rollback(1); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get("key"); got != "original" {
		t.Fatalf("rollback used escaped snapshot: %q", got)
	}
}

func TestRollbackPrunesFutureVersions(t *testing.T) {
	s := New()
	_ = s.Apply(tx("one", "key", "one", 1))
	s.Snapshot(1)
	_ = s.Apply(tx("two", "key", "two", 1))
	s.Snapshot(2)
	if err := s.Rollback(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Rollback(2); err == nil {
		t.Fatal("future snapshot survived rollback")
	}
}

func TestApplyWithRollbackReportsRestoredRoot(t *testing.T) {
	s := New()
	_ = s.Apply(tx("seed", "key", "before", 1))
	x := NewExecutor(s)
	x.MaxGas = 10
	r, err := x.ApplyWithRollback(1, []chain_domain.Tx{tx("ok", "key", "changed", 1), tx("bad", "other", "x", 11)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Root != s.Root() {
		t.Fatalf("result root %q differs from restored root %q", r.Root, s.Root())
	}
}

func TestCatalogPutDetachesEntries(t *testing.T) {
	c := NewCatalog(2)
	entries := map[string]string{"k": "v"}
	record := SnapshotRecord{ID: "s", Height: 1, Root: "root", Checksum: "sum", Entries: entries}
	if err := c.Put(record); err != nil {
		t.Fatal(err)
	}
	entries["k"] = "changed"
	got, _ := c.Get("s")
	if got.Entries["k"] != "v" {
		t.Fatal("catalog retained caller map")
	}
}

func TestCatalogGetDetachesEntries(t *testing.T) {
	c := NewCatalog(2)
	if err := c.Put(MakeSnapshot("s", 1, map[string]string{"k": "v"})); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Get("s")
	got.Entries["k"] = "changed"
	again, _ := c.Get("s")
	if again.Entries["k"] != "v" {
		t.Fatal("catalog exposed stored map")
	}
}

func TestCatalogListDetachesEntries(t *testing.T) {
	c := NewCatalog(2)
	if err := c.Put(SnapshotRecord{ID: "s", Height: 1, Entries: map[string]string{"k": "v"}}); err != nil {
		t.Fatal(err)
	}
	list := c.List()
	list[0].Entries["k"] = "changed"
	again, _ := c.Get("s")
	if again.Entries["k"] != "v" {
		t.Fatal("catalog list exposed stored map")
	}
}
