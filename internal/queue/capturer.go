package queue

import (
	"sync/atomic"
	"time"

	"github.com/bidshard/parser/internal/metrics"
)

// Capturer enqueues events without blocking the hot path.
type Capturer[T any] struct {
	name    string
	ch      chan T
	dropped atomic.Int64
	prep    func(T) T
	logDrop func(T)
}

func NewCapturer[T any](name string, buffer, defaultBuffer int, prep func(T) T, logDrop func(T)) *Capturer[T] {
	if buffer <= 0 {
		buffer = defaultBuffer
	}
	return &Capturer[T]{
		name:    name,
		ch:      make(chan T, buffer),
		prep:    prep,
		logDrop: logDrop,
	}
}

func (c *Capturer[T]) Events() <-chan T {
	if c == nil {
		return nil
	}
	return c.ch
}

func (c *Capturer[T]) Dropped() int64 {
	if c == nil {
		return 0
	}
	return c.dropped.Load()
}

func (c *Capturer[T]) TryCapture(ev T) {
	if c == nil {
		return
	}
	if c.prep != nil {
		ev = c.prep(ev)
	}
	select {
	case c.ch <- ev:
	default:
		c.dropped.Add(1)
		if c.name != "" {
			metrics.RecordQueueDropped(c.name)
		}
		if c.logDrop != nil {
			c.logDrop(ev)
		}
	}
}

// ZeroTime sets TS when zero; useful for event prep hooks.
func ZeroTime(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts
}
