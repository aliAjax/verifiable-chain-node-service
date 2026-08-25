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
	appended := 0
	for i, entry := range entries {
		id := fmt.Sprintf("audit-batch-%d", i)
		bus.SubscribeBuffer(id, 1)
		event := l.Append(entry.Action, entry.Detail)
		appended++
		if err := bus.Publish(Record{At: event.At, Action: event.Action, Payload: event.Detail, Hash: event.Hash}); err != nil {
			bus.Unsubscribe(id)
			l.rollback(appended)
			return err
		}
		bus.Unsubscribe(id)
	}
	return nil
}

func (l *Log) rollback(n int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n <= 0 || n > len(l.events) {
		return
	}
	orig := len(l.events)
	l.events = l.events[:orig-n]
	if orig == n {
		l.prev = ""
	} else {
		l.prev = l.events[len(l.events)-1].Hash
	}
}
