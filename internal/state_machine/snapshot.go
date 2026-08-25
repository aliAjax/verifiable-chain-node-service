package state_machine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
)

type SnapshotRecord struct {
	ID        string
	Height    chain_domain.Height
	Root      string
	Entries   map[string]string
	Size      int
	Checksum  string
	CreatedAt time.Time
	Verified  bool
}
type Catalog struct {
	mu    sync.RWMutex
	items map[string]SnapshotRecord
	max   int
}

func NewCatalog(max int) *Catalog { return &Catalog{items: map[string]SnapshotRecord{}, max: max} }
func (c *Catalog) Put(s SnapshotRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.max {
		return errors.New("snapshot catalog full")
	}
	c.items[s.ID] = s
	return nil
}
func (c *Catalog) Get(id string) (SnapshotRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.items[id]
	return s, ok
}
func (c *Catalog) List() []SnapshotRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o := make([]SnapshotRecord, 0, len(c.items))
	for _, s := range c.items {
		o = append(o, s)
	}
	sort.Slice(o, func(i, j int) bool { return o[i].Height > o[j].Height })
	return o
}
func (c *Catalog) Verify(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.items[id]
	if !ok {
		return false
	}
	s.Verified = checksum(s.Entries) == s.Checksum
	c.items[id] = s
	return s.Verified
}
func checksum(d map[string]string) string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(d[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
func MakeSnapshot(id string, h chain_domain.Height, d map[string]string) SnapshotRecord {
	c := map[string]string{}
	for k, v := range d {
		c[k] = v
	}
	return SnapshotRecord{ID: id, Height: h, Root: checksum(c), Entries: c, Size: len(c), Checksum: checksum(c), CreatedAt: time.Now().UTC()}
}
func (s SnapshotRecord) Valid() bool {
	return s.ID != "" && s.Height > 0 && s.Root != "" && s.Checksum != ""
}
func (s SnapshotRecord) Keys() []string {
	keys := make([]string, 0, len(s.Entries))
	for k := range s.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (s SnapshotRecord) Read(k string) (string, bool) { v, ok := s.Entries[k]; return v, ok }
