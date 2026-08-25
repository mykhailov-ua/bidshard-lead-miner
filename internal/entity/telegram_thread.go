package entity

import (
	"strings"
	"sync"
)

const DefaultTelegramThreadWindow = 6

// ThreadBuffer groups recent messages per channel+author for entity sighting text.
type ThreadBuffer struct {
	mu      sync.Mutex
	max     int
	buckets map[string][]string
}

func NewThreadBuffer(max int) *ThreadBuffer {
	if max <= 0 {
		max = DefaultTelegramThreadWindow
	}
	return &ThreadBuffer{max: max, buckets: make(map[string][]string)}
}

func (b *ThreadBuffer) key(source, author string) string {
	return strings.ToLower(strings.TrimSpace(source)) + "|" + strings.ToLower(strings.TrimSpace(author))
}

func (b *ThreadBuffer) Add(source, author, text string) []string {
	if b == nil {
		return nil
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	k := b.key(source, author)
	win := append(b.buckets[k], text)
	if len(win) > b.max {
		win = win[len(win)-b.max:]
	}
	b.buckets[k] = win
	out := append([]string(nil), win...)
	return out
}

// BundleThreadMessages joins messages oldest-first; delimiter keeps Gemini sighting boundaries visible.
func BundleThreadMessages(msgs []string, max int) string {
	if len(msgs) == 0 {
		return ""
	}
	if max <= 0 {
		max = DefaultTelegramThreadWindow
	}
	if len(msgs) > max {
		msgs = msgs[len(msgs)-max:]
	}
	return strings.Join(msgs, "\n---\n")
}
