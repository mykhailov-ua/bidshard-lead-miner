package warmpath

import (
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/pretty"
	"github.com/bidshard/parser/internal/queue"
)

// Capturer enqueues accepted leads without blocking the hot path.
type Capturer = queue.Capturer[Event]

func NewCapturer(buffer int) *Capturer {
	return queue.NewCapturer("warm", buffer, 256, prepareEvent, logDroppedEvent)
}

func prepareEvent(ev Event) Event {
	if ev.TS.IsZero() {
		ev.TS = queue.ZeroTime(ev.TS)
	}
	ev.Snippet = pretty.Truncate(strings.Join(strings.Fields(ev.Snippet), " "), 500)
	return ev
}

func logDroppedEvent(ev Event) {
	slog.Debug("warm path queue full, dropping lead", "hash_id", ev.HashID, "source", ev.Source)
}
