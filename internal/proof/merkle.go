package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
)

type Node struct {
	Hash        string
	Left, Right *Node
	Key, Value  string
}
type Step struct {
	Hash string
	Left bool
}
type MerkleProof struct {
	Key, Value, Root string
	Steps            []Step
	Version          uint32
}

func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func BuildTree(items map[string]string) *Node {
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	nodes := []*Node{}
	for _, k := range keys {
		nodes = append(nodes, &Node{Hash: hash(k + "=" + items[k]), Key: k, Value: items[k]})
	}
	if len(nodes) == 0 {
		return &Node{Hash: hash("")}
	}
	for len(nodes) > 1 {
		next := []*Node{}
		for i := 0; i < len(nodes); i += 2 {
			j := i + 1
			if j == len(nodes) {
				j = i
			}
			next = append(next, &Node{Hash: hash(nodes[i].Hash + nodes[j].Hash), Left: nodes[i], Right: nodes[j]})
		}
		nodes = next
	}
	return nodes[0]
}
func Root(items map[string]string) string { return BuildTree(items).Hash }

func Proof(items map[string]string, key string) (MerkleProof, error) {
	if _, ok := items[key]; !ok {
		return MerkleProof{}, errors.New("key not found")
	}
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	index := sort.SearchStrings(keys, key)
	hashes := make([]string, 0, len(keys))
	for _, k := range keys {
		hashes = append(hashes, hash(k+"="+items[k]))
	}
	steps := []Step{}
	for len(hashes) > 1 {
		if len(hashes)%2 == 1 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		sibling := index ^ 1
		steps = append(steps, Step{Hash: hashes[sibling], Left: sibling < index})
		next := make([]string, 0, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			next = append(next, hash(hashes[i]+hashes[i+1]))
		}
		index /= 2
		hashes = next
	}
	return MerkleProof{Key: key, Value: items[key], Root: hashes[0], Steps: steps, Version: 1}, nil
}
func VerifyProof(p MerkleProof) bool {
	if p.Key == "" || p.Root == "" || p.Version == 0 {
		return false
	}
	current := hash(p.Key + "=" + p.Value)
	for _, step := range p.Steps {
		if step.Hash == "" {
			return false
		}
		if step.Left {
			current = hash(step.Hash + current)
		} else {
			current = hash(current + step.Hash)
		}
	}
	return current == p.Root
}
func Explain(p MerkleProof) map[string]interface{} {
	return map[string]interface{}{"key": p.Key, "root": p.Root, "steps": len(p.Steps), "algorithm": "merkle-sha256", "verified": VerifyProof(p)}
}
func (n *Node) Leaves(out *[]Node) {
	if n == nil {
		return
	}
	if n.Left == nil && n.Right == nil {
		*out = append(*out, *n)
		return
	}
	n.Left.Leaves(out)
	if n.Right != nil && n.Right != n.Left {
		n.Right.Leaves(out)
	}
}
