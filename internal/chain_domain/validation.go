package chain_domain

import (
	"errors"
	"math"
	"strings"
)

func ValidateTx(t Tx) error {
	if strings.TrimSpace(t.From) == "" {
		return errors.New("from required")
	}
	if strings.TrimSpace(t.To) == "" {
		return errors.New("to required")
	}
	if t.Gas == 0 || t.Gas > math.MaxUint32 {
		return errors.New("invalid gas")
	}
	if len(t.Key) > 256 || len(t.Value) > 4096 {
		return errors.New("key/value too large")
	}
	if t.Nonce == math.MaxUint64 {
		return errors.New("nonce overflow")
	}
	return nil
}
func ValidateNetwork(n Network) error {
	if n.ID == "" || n.ChainID == "" {
		return errors.New("network identity required")
	}
	if len(n.Validators) == 0 {
		return errors.New("validator set empty")
	}
	var total uint64
	for _, v := range n.Validators {
		if v.Address == "" || v.Power == 0 {
			return errors.New("invalid validator")
		}
		if math.MaxUint64-total < v.Power {
			return errors.New("power overflow")
		}
		total += v.Power
	}
	return nil
}
func Quorum(v []Validator) uint64 {
	var p uint64
	for _, x := range v {
		p += x.Power
	}
	return (p*2)/3 + 1
}
func HasQuorum(v []Validator, votes []Vote) bool {
	powers := map[string]uint64{}
	for _, x := range v {
		powers[x.Address] = x.Power
	}
	var got uint64
	seen := map[string]bool{}
	for _, x := range votes {
		if !seen[x.Validator] {
			got += powers[x.Validator]
			seen[x.Validator] = true
		}
	}
	return got >= Quorum(v)
}
