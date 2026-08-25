package codec

import (
	"errors"
	"fmt"
	"math"
)

type Limits struct {
	MaxBytes  int
	MaxString int
	MaxList   int
	MaxDepth  int
	MaxMap    int
}

func DefaultLimits() Limits {
	return Limits{MaxBytes: 2 << 20, MaxString: 65535, MaxList: 4096, MaxDepth: 64, MaxMap: 100000}
}
func (l Limits) CheckString(s string) error {
	if len(s) > l.MaxString {
		return errors.New("string limit exceeded")
	}
	return nil
}
func (l Limits) CheckBytes(b []byte) error {
	if len(b) > l.MaxBytes {
		return errors.New("byte limit exceeded")
	}
	return nil
}
func (l Limits) CheckList(n int) error {
	if n < 0 || n > l.MaxList {
		return errors.New("list limit exceeded")
	}
	return nil
}
func (l Limits) CheckDepth(n int) error {
	if n < 0 || n > l.MaxDepth {
		return errors.New("depth limit exceeded")
	}
	return nil
}
func (l Limits) CheckMap(n int) error {
	if n < 0 || n > l.MaxMap {
		return errors.New("map limit exceeded")
	}
	return nil
}
func safeAdd(a, b uint64) (uint64, error) {
	if math.MaxUint64-a < b {
		return 0, errors.New("integer overflow")
	}
	return a + b, nil
}
func safeMul(a, b uint64) (uint64, error) {
	if a != 0 && b > math.MaxUint64/a {
		return 0, errors.New("integer overflow")
	}
	return a * b, nil
}
func encodeVersion(v uint32) string { return fmt.Sprintf("v%d", v) }
func negotiate(local, remote []uint32) uint32 {
	for i := len(local) - 1; i >= 0; i-- {
		for _, r := range remote {
			if local[i] == r {
				return r
			}
		}
	}
	return 0
}
func versionAllowed(v uint32, allowed []uint32) bool {
	for _, x := range allowed {
		if x == v {
			return true
		}
	}
	return false
}
func checksum(data []byte) uint32 {
	var h uint32 = 2166136261
	for _, b := range data {
		h ^= uint32(b)
		h *= 16777619
	}
	return h
}
func frame(payload []byte, l Limits) ([]byte, error) {
	if e := l.CheckBytes(payload); e != nil {
		return nil, e
	}
	out := make([]byte, 4+len(payload))
	out[0] = byte(len(payload) >> 24)
	out[1] = byte(len(payload) >> 16)
	out[2] = byte(len(payload) >> 8)
	out[3] = byte(len(payload))
	copy(out[4:], payload)
	return out, nil
}
func unframe(data []byte, l Limits) ([]byte, error) {
	if len(data) < 4 {
		return nil, ErrMalformed
	}
	n := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if n != len(data)-4 {
		return nil, ErrMalformed
	}
	if e := l.CheckBytes(data[4:]); e != nil {
		return nil, e
	}
	return data[4:], nil
}
