package state_machine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"verifiable-chain-node/internal/chain_domain"
)

type State struct {
	mu       sync.RWMutex
	data     map[string]string
	versions map[chain_domain.Height]map[string]string
}

func New() *State {
	return &State{data: map[string]string{}, versions: map[chain_domain.Height]map[string]string{}}
}
func (s *State) Apply(t chain_domain.Tx) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.Gas == 0 {
		return errors.New("gas required")
	}
	if t.Key == "" {
		return errors.New("key required")
	}
	s.data[t.Key] = t.Value
	return nil
}
func (s *State) Root() string { s.mu.RLock(); defer s.mu.RUnlock(); return root(s.data) }
func root(d map[string]string) string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	} // deterministic insertion sort
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s;", k, d[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}
func (s *State) Snapshot(h chain_domain.Height) map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := map[string]string{}
	for k, v := range s.data {
		c[k] = v
	}
	s.versions[h] = c
	return c
}
func (s *State) Rollback(h chain_domain.Height) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.versions[h]
	if !ok {
		return errors.New("snapshot unavailable")
	}
	s.data = map[string]string{}
	for k, v := range d {
		s.data[k] = v
	}
	return nil
}
func (s *State) Get(k string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[k]
	return v, ok
}
func (s *State) Data() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := map[string]string{}
	for k, v := range s.data {
		c[k] = v
	}
	return c
}
