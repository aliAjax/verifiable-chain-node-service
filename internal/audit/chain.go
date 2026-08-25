package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Chain struct {
	mu      sync.RWMutex
	records []Record
	secret  string
}
type Record struct {
	Sequence                                                    uint64
	At                                                          time.Time
	Actor, Action, Resource, Payload, PrevHash, Hash, Signature string
}

func NewChain(secret string) *Chain { return &Chain{secret: secret} }
func (c *Chain) Append(actor, action, resource, payload string) Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := ""
	seq := uint64(1)
	if len(c.records) > 0 {
		prev = c.records[len(c.records)-1].Hash
		seq = c.records[len(c.records)-1].Sequence + 1
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", seq, actor, action, resource, payload, prev, c.secret)))
	r := Record{Sequence: seq, At: time.Now().UTC(), Actor: actor, Action: action, Resource: resource, Payload: payload, PrevHash: prev, Hash: hex.EncodeToString(h[:]), Signature: hex.EncodeToString(h[:])}
	c.records = append(c.records, r)
	return r
}
func (c *Chain) Verify() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prev := ""
	for i, r := range c.records {
		if r.Sequence != uint64(i+1) {
			return errors.New("sequence gap")
		}
		if r.PrevHash != prev {
			return errors.New("audit link mismatch")
		}
		h := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", r.Sequence, r.Actor, r.Action, r.Resource, r.Payload, r.PrevHash, c.secret)))
		expected := hex.EncodeToString(h[:])
		if r.Hash != expected || r.Signature != expected {
			return errors.New("audit digest mismatch")
		}
		prev = r.Hash
	}
	return nil
}
func (c *Chain) List() []Record {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Record(nil), c.records...)
}
func (c *Chain) Since(seq uint64) []Record {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o := []Record{}
	for _, r := range c.records {
		if r.Sequence > seq {
			o = append(o, r)
		}
	}
	return o
}
func (r Record) Digest() string {
	h := sha256.Sum256([]byte(r.Hash + r.Signature))
	return hex.EncodeToString(h[:])
}
func (r Record) Valid() bool {
	return r.Sequence > 0 && !r.At.IsZero() && r.Hash != "" && r.Signature != ""
}
func (r Record) String() string {
	return fmt.Sprintf("%d %s %s %s", r.Sequence, r.Actor, r.Action, r.Resource)
}

type Subscriber struct {
	ID     string
	Ch     chan Record
	Closed bool
}
type Bus struct {
	mu   sync.Mutex
	subs map[string]Subscriber
	peak int
}

func NewBus() *Bus { return &Bus{subs: map[string]Subscriber{}} }
func (b *Bus) Subscribe(id string) chan Record {
	return b.SubscribeBuffer(id, 64)
}
func (b *Bus) SubscribeBuffer(id string, size int) chan Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Record, size)
	b.subs[id] = Subscriber{ID: id, Ch: ch}
	if len(b.subs) > b.peak {
		b.peak = len(b.subs)
	}
	return ch
}
func (b *Bus) Publish(r Record) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, s := range b.subs {
		select {
		case s.Ch <- r:
		default:
			close(s.Ch)
			delete(b.subs, id)
			return fmt.Errorf("subscriber %s: %w", id, ErrDelivery)
		}
	}
	return nil
}
func (b *Bus) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if s, ok := b.subs[id]; ok {
		close(s.Ch)
		delete(b.subs, id)
	}
}
func (b *Bus) ActiveSubscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}
func (b *Bus) PeakSubscribers() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.peak
}

var ErrDelivery = errors.New("audit delivery failed")

type RecordInput struct {
	Actor, Action, Resource, Payload string
}

func (c *Chain) AppendBatch(bus *Bus, inputs []RecordInput) error {
	appended := 0
	for _, input := range inputs {
		record := c.Append(input.Actor, input.Action, input.Resource, input.Payload)
		appended++
		if err := bus.Publish(record); err != nil {
			c.rollback(appended)
			return err
		}
	}
	return nil
}

func (c *Chain) rollback(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n <= 0 || n > len(c.records) {
		return
	}
	c.records = c.records[:len(c.records)-n]
}
