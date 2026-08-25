package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Entry struct {
	At      time.Time              `json:"at"`
	Level   Level                  `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
	TraceID string                 `json:"trace_id,omitempty"`
}
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	min    Level
	fields map[string]interface{}
}

func NewLogger(out io.Writer, min Level) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{out: out, min: min, fields: map[string]interface{}{}}
}
func (l *Logger) With(k string, v interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	f := map[string]interface{}{}
	for x, y := range l.fields {
		f[x] = y
	}
	f[k] = v
	return &Logger{out: l.out, min: l.min, fields: f}
}
func rank(x Level) int {
	switch x {
	case LevelDebug:
		return 1
	case LevelInfo:
		return 2
	case LevelWarn:
		return 3
	case LevelError:
		return 4
	}
	return 2
}
func (l *Logger) Log(ctx context.Context, lv Level, msg string, fields map[string]interface{}) {
	if rank(lv) < rank(l.min) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f := map[string]interface{}{}
	for k, v := range l.fields {
		f[k] = v
	}
	for k, v := range fields {
		f[k] = v
	}
	e := Entry{At: time.Now().UTC(), Level: lv, Message: msg, Fields: f}
	if id, ok := ctx.Value("trace_id").(string); ok {
		e.TraceID = id
	}
	_ = json.NewEncoder(l.out).Encode(e)
}
func (l *Logger) Debug(c context.Context, m string, f map[string]interface{}) {
	l.Log(c, LevelDebug, m, f)
}
func (l *Logger) Info(c context.Context, m string, f map[string]interface{}) {
	l.Log(c, LevelInfo, m, f)
}
func (l *Logger) Warn(c context.Context, m string, f map[string]interface{}) {
	l.Log(c, LevelWarn, m, f)
}
func (l *Logger) Error(c context.Context, m string, f map[string]interface{}) {
	l.Log(c, LevelError, m, f)
}

type Span struct {
	TraceID string
	Name    string
	Started time.Time
	Attrs   map[string]string
	Err     error
}

func StartSpan(name string) Span {
	return Span{TraceID: fmt.Sprintf("%d", time.Now().UnixNano()), Name: name, Started: time.Now(), Attrs: map[string]string{}}
}
func (s *Span) Set(k, v string)             { s.Attrs[k] = v }
func (s *Span) End(err error) time.Duration { s.Err = err; return time.Since(s.Started) }
func (s Span) Duration() time.Duration      { return time.Since(s.Started) }
func (s Span) Valid() bool                  { return s.Name != "" && s.TraceID != "" }

type Counter struct {
	mu     sync.Mutex
	name   string
	value  uint64
	labels map[string]string
}

func NewCounter(n string) *Counter   { return &Counter{name: n, labels: map[string]string{}} }
func (c *Counter) Add(n uint64)      { c.mu.Lock(); defer c.mu.Unlock(); c.value += n }
func (c *Counter) Value() uint64     { c.mu.Lock(); defer c.mu.Unlock(); return c.value }
func (c *Counter) Label(k, v string) { c.mu.Lock(); defer c.mu.Unlock(); c.labels[k] = v }
func (c *Counter) Snapshot() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	labels := make(map[string]string, len(c.labels))
	for k, v := range c.labels {
		labels[k] = v
	}
	return map[string]interface{}{"name": c.name, "value": c.value, "labels": labels}
}
