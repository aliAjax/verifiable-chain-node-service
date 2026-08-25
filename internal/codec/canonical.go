package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"verifiable-chain-node/internal/chain_domain"
)

const MaxBlockBytes = 2 << 20

var ErrMalformed = errors.New("malformed canonical encoding")

func put(w *bytes.Buffer, s string) error {
	if len(s) > 65535 {
		return ErrMalformed
	}
	var x [2]byte
	binary.BigEndian.PutUint16(x[:], uint16(len(s)))
	w.Write(x[:])
	w.WriteString(s)
	return nil
}
func get(r *bytes.Reader) (string, error) {
	var x uint16
	if binary.Read(r, binary.BigEndian, &x) != nil {
		return "", ErrMalformed
	}
	if int(x) > r.Len() {
		return "", ErrMalformed
	}
	b := make([]byte, x)
	if _, e := io.ReadFull(r, b); e != nil {
		return "", e
	}
	return string(b), nil
}
func EncodeTx(t chain_domain.Tx) ([]byte, error) {
	var b bytes.Buffer
	for _, s := range []string{t.ID, t.From, t.To, t.Key, t.Value, t.Signature} {
		if e := put(&b, s); e != nil {
			return nil, e
		}
	}
	for _, n := range []uint64{t.Nonce, t.Fee, t.Gas, uint64(t.ExpiresAt), uint64(t.Size)} {
		binary.Write(&b, binary.BigEndian, n)
	}
	if b.Len() > MaxBlockBytes {
		return nil, ErrMalformed
	}
	return b.Bytes(), nil
}
func DecodeTx(data []byte) (chain_domain.Tx, error) {
	if len(data) > MaxBlockBytes {
		return chain_domain.Tx{}, ErrMalformed
	}
	r := bytes.NewReader(data)
	vals := make([]string, 6)
	for i := range vals {
		s, e := get(r)
		if e != nil {
			return chain_domain.Tx{}, e
		}
		vals[i] = s
	}
	nums := make([]uint64, 5)
	for i := range nums {
		if binary.Read(r, binary.BigEndian, &nums[i]) != nil {
			return chain_domain.Tx{}, ErrMalformed
		}
	}
	return chain_domain.Tx{ID: vals[0], From: vals[1], To: vals[2], Key: vals[3], Value: vals[4], Signature: vals[5], Nonce: nums[0], Fee: nums[1], Gas: nums[2], ExpiresAt: int64(nums[3]), Size: uint32(nums[4])}, nil
}
func EncodeBlock(b chain_domain.Block) ([]byte, error) {
	var out bytes.Buffer
	h := b.Header
	for _, s := range []string{b.Hash, h.PrevHash, h.StateRoot, h.TxRoot, h.Proposer} {
		if e := put(&out, s); e != nil {
			return nil, e
		}
	}
	for _, n := range []uint64{uint64(h.Height), uint64(h.View), uint64(h.Timestamp), uint64(h.Version), uint64(len(b.Transactions))} {
		binary.Write(&out, binary.BigEndian, n)
	}
	for _, t := range b.Transactions {
		d, e := EncodeTx(t)
		if e != nil {
			return nil, e
		}
		binary.Write(&out, binary.BigEndian, uint32(len(d)))
		out.Write(d)
	}
	if out.Len() > MaxBlockBytes {
		return nil, fmt.Errorf("block too large")
	}
	return out.Bytes(), nil
}
