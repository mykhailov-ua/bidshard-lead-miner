package coldpath

import (
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// Capturer enqueues junk events without blocking the hot path.
type Capturer struct {
	ch      chan Event
	dropped atomic.Int64
}

func NewCapturer(buffer int) *Capturer {
	if buffer <= 0 {
		buffer = 512
	}
	return &Capturer{ch: make(chan Event, buffer)}
}

func (c *Capturer) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.ch
}

func (c *Capturer) Dropped() int64 {
	if c == nil {
		return 0
	}
	return c.dropped.Load()
}

func (c *Capturer) TryCapture(ev Event) {
	if c == nil {
		return
	}
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	ev.Snippet = truncate(strings.Join(strings.Fields(ev.Snippet), " "), 500)
	ev.ContactHint = maskContactHint(ev.ContactHint)
	select {
	case c.ch <- ev:
	default:
		c.dropped.Add(1)
		slog.Debug("junk queue full, dropping event", "reason", ev.Reason, "source", ev.Source)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func maskContactHint(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "@") {
		return maskEmail(v)
	}
	if strings.HasPrefix(v, "@") {
		if len(v) <= 2 {
			return "@***"
		}
		return v[:2] + "***"
	}
	if len(v) <= 3 {
		return "***"
	}
	return v[:1] + "***"
}

func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local := email[:at]
	domain := email[at+1:]
	if len(local) == 0 {
		return "***@" + domain
	}
	return local[:1] + "***@" + domain
}
