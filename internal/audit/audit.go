package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type Event struct {
	At                   time.Time
	Action, Detail, Hash string
}
type Log struct {
	mu     sync.Mutex
	prev   string
	events []Event
}

func New() *Log { return &Log{} }
func (l *Log) Append(a, d string) Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	h := sha256.Sum256([]byte(l.prev + fmt.Sprint(a, d, time.Now().UnixNano())))
	e := Event{At: time.Now(), Action: a, Detail: d, Hash: hex.EncodeToString(h[:])}
	l.prev = e.Hash
	l.events = append(l.events, e)
	return e
}
func (l *Log) Events() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Event(nil), l.events...)
}
