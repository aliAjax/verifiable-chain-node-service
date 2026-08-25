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
type Entry struct {
	Action string
	Detail string
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
func (l *Log) AppendBatch(bus *Bus, entries []Entry) error {
	for i, entry := range entries {
		id := fmt.Sprintf("audit-batch-%d", i)
		bus.SubscribeBuffer(id, 1)
		defer bus.Unsubscribe(id)
		event := l.Append(entry.Action, entry.Detail)
		if err := bus.Publish(Record{At: event.At, Action: event.Action, Payload: event.Detail, Hash: event.Hash}); err != nil {
			return err
		}
	}
	return nil
}
