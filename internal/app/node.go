package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"verifiable-chain-node/internal/audit"
	"verifiable-chain-node/internal/chain_domain"
	"verifiable-chain-node/internal/consensus"
	"verifiable-chain-node/internal/mempool"
	"verifiable-chain-node/internal/observability"
	"verifiable-chain-node/internal/p2p"
	"verifiable-chain-node/internal/proof"
	"verifiable-chain-node/internal/repository"
	"verifiable-chain-node/internal/state_machine"
)

type Config struct {
	Network chain_domain.Network
	Addr    string
}

func DefaultConfig() Config {
	return Config{Network: chain_domain.Network{ID: "local", Name: "Verifiable Chain", ChainID: "vchain-1", Version: 1, Validators: []chain_domain.Validator{{Address: "validator-1", Power: 1}, {Address: "validator-2", Power: 1}, {Address: "validator-3", Power: 1}}}, Addr: ":8080"}
}

type Node struct {
	cfg         Config
	lifecycle   context.Context
	pool        *mempool.Pool
	state       *state_machine.State
	repo        *repository.Repo
	peers       *p2p.Manager
	bft         *consensus.Engine
	audit       *audit.Log
	metrics     *observability.Metrics
	checkpoints []chain_domain.Block
	opMu        sync.Mutex
}

func NewNode(c Config) *Node {
	return NewNodeWithContext(context.Background(), c)
}
func NewNodeWithContext(_ context.Context, c Config) *Node {
	n := &Node{cfg: c, lifecycle: context.Background(), pool: mempool.New(1000), state: state_machine.New(), repo: repository.New(), peers: p2p.New(), bft: consensus.New(c.Network.Validators), audit: audit.New(), metrics: observability.New()}
	n.peers.Add(p2p.Peer{ID: "validator-1", Address: "127.0.0.1:9001", Score: 100})
	return n
}
func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", n.health)
	mux.HandleFunc("/metrics", n.metricsH)
	mux.HandleFunc("/api/v1/networks", n.networks)
	mux.HandleFunc("/api/v1/peers", n.peerH)
	mux.HandleFunc("/api/v1/transactions", n.txH)
	mux.HandleFunc("/api/v1/mempool", n.mempoolH)
	mux.HandleFunc("/api/v1/blocks/", n.blockH)
	mux.HandleFunc("/api/v1/blocks/mine", n.mine)
	mux.HandleFunc("/api/v1/sync", n.syncH)
	mux.HandleFunc("/api/v1/snapshots", n.snapH)
	mux.HandleFunc("/api/v1/snapshots/", n.snapVerify)
	mux.HandleFunc("/api/v1/rollback-plan", n.rollback)
	mux.HandleFunc("/api/v1/light/checkpoints", n.checkpointsH)
	mux.HandleFunc("/api/v1/proofs/", n.proofH)
	mux.HandleFunc("/api/v1/consensus/views", n.consensusH)
	return n.middleware(mux)
}
func (n *Node) middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.metrics.Requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r.WithContext(context.Background()))
	})
}
func write(w http.ResponseWriter, v interface{}, code int) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func (n *Node) health(w http.ResponseWriter, r *http.Request) {
	write(w, map[string]interface{}{"status": "ok", "network": n.cfg.Network.ChainID, "height": n.repo.Latest().Header.Height}, 200)
}
func (n *Node) metricsH(w http.ResponseWriter, r *http.Request) { write(w, n.metrics.Snapshot(), 200) }
func (n *Node) networks(w http.ResponseWriter, r *http.Request) {
	write(w, []chain_domain.Network{n.cfg.Network}, 200)
}
func (n *Node) peerH(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var p p2p.Peer
		if json.NewDecoder(r.Body).Decode(&p) != nil {
			write(w, map[string]string{"error": "bad json"}, 400)
			return
		}
		if p.ID == "" {
			write(w, map[string]string{"error": "id required"}, 400)
			return
		}
		n.peers.Add(p)
		write(w, p, 201)
		return
	}
	write(w, n.peers.List(), 200)
}
func (n *Node) txH(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		write(w, map[string]string{"error": "method"}, 405)
		return
	}
	var t chain_domain.Tx
	if json.NewDecoder(r.Body).Decode(&t) != nil {
		write(w, map[string]string{"error": "bad json"}, 400)
		return
	}
	if t.ID == "" {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		t.ID = hex.EncodeToString(b)
	}
	if e := n.pool.Add(t); e != nil {
		n.metrics.Errors.Add(1)
		write(w, map[string]string{"error": e.Error()}, 409)
		return
	}
	n.audit.Append("tx.accept", t.ID)
	write(w, t, 202)
}
func (n *Node) mempoolH(w http.ResponseWriter, r *http.Request) { write(w, n.pool.List(), 200) }
func (n *Node) blockH(w http.ResponseWriter, r *http.Request) {
	h, e := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/v1/blocks/"), 10, 64)
	if e != nil {
		write(w, map[string]string{"error": "height"}, 400)
		return
	}
	b, ok := n.repo.Block(chain_domain.Height(h))
	if !ok {
		write(w, map[string]string{"error": "not found"}, 404)
		return
	}
	write(w, b, 200)
}
func (n *Node) mine(w http.ResponseWriter, r *http.Request) {
	n.opMu.Lock()
	defer n.opMu.Unlock()
	txs := n.pool.List()
	prev := n.repo.Latest()
	h := prev.Header.Height + 1
	for _, t := range txs {
		if e := n.state.Apply(t); e != nil {
			continue
		}
	}
	root := n.state.Root()
	b := chain_domain.Block{Header: chain_domain.Header{Height: h, View: 0, PrevHash: prev.Hash, StateRoot: root, TxRoot: chain_domain.MerkleRoot(txs), Proposer: "validator-1", Timestamp: time.Now().Unix(), Version: 1}, Transactions: txs}
	b.Hash = b.ComputeHash()
	if e := n.repo.PutBlock(b); e != nil {
		write(w, map[string]string{"error": e.Error()}, 409)
		return
	}
	n.pool.Remove(func() []string {
		ids := []string{}
		for _, t := range txs {
			ids = append(ids, t.ID)
		}
		return ids
	}())
	n.metrics.Blocks.Add(1)
	n.audit.Append("block.commit", fmt.Sprint(h))
	write(w, b, 201)
}
func (n *Node) syncH(w http.ResponseWriter, r *http.Request) {
	write(w, map[string]interface{}{"status": "accepted", "from": n.repo.Latest().Header.Height, "peers": len(n.peers.List())}, 202)
}
func (n *Node) snapH(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		write(w, n.repo.Snapshots(), 200)
		return
	}
	n.opMu.Lock()
	defer n.opMu.Unlock()
	id := fmt.Sprintf("snap-%d", time.Now().UnixNano())
	s := chain_domain.Snapshot{ID: id, Height: n.repo.Latest().Header.Height, Root: n.state.Root(), Data: n.state.Data(), CreatedAt: time.Now()}
	n.repo.PutSnapshot(s)
	n.state.Snapshot(s.Height)
	n.audit.Append("snapshot.create", id)
	write(w, s, 201)
}
func (n *Node) snapVerify(w http.ResponseWriter, r *http.Request) {
	n.opMu.Lock()
	defer n.opMu.Unlock()
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/snapshots/")
	id = strings.TrimSuffix(id, "/verify")
	if id == "" {
		write(w, map[string]string{"error": "snapshot id required"}, 400)
		return
	}
	s, ok := n.repo.Snapshot(id)
	if !ok {
		write(w, map[string]string{"error": "not found"}, 404)
		return
	}
	s.Verified = s.Root == n.state.Root()
	n.repo.PutSnapshot(s)
	write(w, map[string]interface{}{"id": id, "verified": s.Verified}, 200)
}
func (n *Node) rollback(w http.ResponseWriter, r *http.Request) {
	write(w, map[string]interface{}{"latest": n.repo.Latest().Header.Height, "action": "rollback requires verified snapshot"}, 200)
}
func (n *Node) checkpointsH(w http.ResponseWriter, r *http.Request) { write(w, n.checkpoints, 200) }
func (n *Node) proofH(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/api/v1/proofs/")
	write(w, proof.Build(n.state, key), 200)
}
func (n *Node) consensusH(w http.ResponseWriter, r *http.Request) { write(w, n.bft.Status(), 200) }
