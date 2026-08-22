package coldpath

import (
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
	"github.com/bidshard/parser/internal/queue"
)

// Capturer enqueues junk events without blocking the hot path.
type Capturer = queue.Capturer[Event]

func NewCapturer(buffer int) *Capturer {
	return queue.NewCapturer("junk", buffer, 512, prepareEvent, logDroppedEvent)
}

func prepareEvent(ev Event) Event {
	if ev.TS.IsZero() {
		ev.TS = queue.ZeroTime(ev.TS)
	}
	ev.Snippet = pretty.Truncate(strings.Join(strings.Fields(ev.Snippet), " "), 500)
	ev.ContactHint = maskContactHint(ev.ContactHint)
	return ev
}

func logDroppedEvent(ev Event) {
	slog.Debug("junk queue full, dropping event", "reason", ev.Reason, "source", ev.Source)
}
