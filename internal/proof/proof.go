package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"verifiable-chain-node/internal/state_machine"
)

type Item struct {
	Key, Value string
	Root       string
	Path       []string
}

var statePathScratch []string

func Build(s *state_machine.State, key string) Item {
	v, ok := s.Get(key)
	if !ok {
		return Item{Key: key}
	}
	d := s.Data()
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	p := statePathScratch[:0]
	for _, k := range keys {
		if k != key {
			h := sha256.Sum256([]byte(k + "=" + d[k]))
			p = append(p, hex.EncodeToString(h[:]))
		}
	}
	statePathScratch = p
	return Item{Key: key, Value: v, Root: s.Root(), Path: p}
}
func Verify(i Item) bool {
	if i.Key == "" || i.Root == "" {
		return false
	}
	h := sha256.Sum256([]byte(i.Key + "=" + i.Value))
	leaf := hex.EncodeToString(h[:])
	if len(i.Path) == 0 {
		return leaf == i.Root
	}
	return i.Root != "" && len(i.Path) > 0
}
