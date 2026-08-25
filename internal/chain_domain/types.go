package chain_domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"time"
)

type Height uint64
type View uint64

type Validator struct {
	Address   string `json:"address"`
	Power     uint64 `json:"power"`
	PublicKey string `json:"public_key"`
}
type Network struct {
	ID, Name, ChainID string
	Version           uint32
	Validators        []Validator
}
type Tx struct {
	ID, From, To    string
	Nonce, Fee, Gas uint64
	Key, Value      string
	ExpiresAt       int64
	Size            uint32
	Signature       string
}
type Header struct {
	Height                                Height `json:"height"`
	View                                  View   `json:"view"`
	PrevHash, StateRoot, TxRoot, Proposer string
	Timestamp                             int64
	Version                               uint32
}
type Block struct {
	Header       Header `json:"header"`
	Transactions []Tx   `json:"transactions"`
	Commit       []Vote `json:"commit"`
	Hash         string `json:"hash"`
}
type Vote struct {
	Validator string `json:"validator"`
	Height    Height `json:"height"`
	View      View   `json:"view"`
	Type      string `json:"type"`
	Signature string `json:"signature"`
}
type Snapshot struct {
	ID        string            `json:"id"`
	Height    Height            `json:"height"`
	Root      string            `json:"root"`
	Data      map[string]string `json:"data"`
	CreatedAt time.Time         `json:"created_at"`
	Verified  bool              `json:"verified"`
}

func (h Header) Bytes() []byte {
	return []byte(fmt.Sprintf("%d|%d|%s|%s|%s|%s|%d|%d", h.Height, h.View, h.PrevHash, h.StateRoot, h.TxRoot, h.Proposer, h.Timestamp, h.Version))
}
func (h Header) Hash() string { s := sha256.Sum256(h.Bytes()); return hex.EncodeToString(s[:]) }
func TxHash(t Tx) string {
	s := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d|%d|%d|%s|%s", t.ID, t.From, t.To, t.Nonce, t.Fee, t.Gas, t.Key, t.Value)))
	return hex.EncodeToString(s[:])
}
func MerkleRoot(txs []Tx) string {
	if len(txs) == 0 {
		return sha256Hex(nil)
	}
	a := make([]string, len(txs))
	for i, t := range txs {
		a[i] = TxHash(t)
	}
	for len(a) > 1 {
		var n []string
		for i := 0; i < len(a); i += 2 {
			j := i + 1
			if j == len(a) {
				j = i
			}
			n = append(n, sha256Hex([]byte(a[i]+a[j])))
		}
		a = n
	}
	return a[0]
}
func sha256Hex(b []byte) string      { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func (b *Block) ComputeHash() string { return b.Header.Hash() }
func ValidateBlock(b Block, prev *Block) error {
	if b.Header.Height == 0 {
		return errors.New("height must be positive")
	}
	if prev != nil {
		if b.Header.Height != prev.Header.Height+1 {
			return errors.New("height discontinuity")
		}
		if b.Header.PrevHash != prev.Hash {
			return errors.New("prev hash mismatch")
		}
	}
	if b.Header.TxRoot != MerkleRoot(b.Transactions) {
		return errors.New("tx root mismatch")
	}
	if b.Hash == "" {
		return errors.New("missing hash")
	}
	return nil
}
func SortValidators(v []Validator) []Validator {
	out := append([]Validator(nil), v...)
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out
}
