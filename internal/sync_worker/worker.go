package sync_worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"verifiable-chain-node/internal/chain_domain"
	"verifiable-chain-node/internal/repository"
)

type Status string

const (
	StatusIdle     Status = "idle"
	StatusRunning  Status = "running"
	StatusFailed   Status = "failed"
	StatusComplete Status = "complete"
)

type Request struct {
	From, To chain_domain.Height
	Peer     string
	MaxBytes int64
}
type Result struct {
	Status                Status
	Downloaded            int
	Verified              int
	Rejected              int
	Error                 string
	StartedAt, FinishedAt time.Time
}
type Worker struct {
	mu     sync.Mutex
	repo   *repository.Repo
	active bool
	last   Result
}

func New(r *repository.Repo) *Worker { return &Worker{repo: r} }
func (w *Worker) Start(ctx context.Context, req Request) Result {
	w.mu.Lock()
	if w.active {
		w.mu.Unlock()
		return Result{Status: StatusFailed, Error: "sync already running"}
	}
	w.active = true
	w.last = Result{Status: StatusRunning, StartedAt: time.Now()}
	w.mu.Unlock()
	defer func() { w.mu.Lock(); w.active = false; w.mu.Unlock() }()
	if req.To < req.From {
		return w.finish(Result{Status: StatusFailed, Error: "invalid range"})
	}
	res := w.last
	for h := req.From; h <= req.To; h++ {
		select {
		case <-ctx.Done():
			res.Status = StatusFailed
			res.Error = ctx.Err().Error()
			return w.finish(res)
		default:
		}
		if req.MaxBytes > 0 && int64(res.Downloaded*1024) > req.MaxBytes {
			res.Status = StatusFailed
			res.Error = "download quota exceeded"
			return w.finish(res)
		}
		if _, ok := w.repo.Block(h); ok {
			res.Verified++
			continue
		}
		res.Downloaded++
		time.Sleep(time.Millisecond)
	}
	res.Status = StatusComplete
	return w.finish(res)
}
func (w *Worker) finish(r Result) Result {
	r.FinishedAt = time.Now()
	w.mu.Lock()
	w.last = r
	w.mu.Unlock()
	return r
}
func (w *Worker) Last() Result { w.mu.Lock(); defer w.mu.Unlock(); return w.last }
func ValidateRange(req Request) error {
	if req.To-req.From > 100000 {
		return errors.New("range too large")
	}
	if req.Peer == "" {
		return errors.New("peer required")
	}
	return nil
}
func Explain(r Result) string {
	if r.Status == StatusComplete {
		return fmt.Sprintf("verified %d blocks", r.Verified)
	}
	return r.Error
}
