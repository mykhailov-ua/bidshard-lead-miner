package telegrambot

import (
	"context"
	"strings"
	"time"
)

const maxTelegramMessageLen = 4096

// SplitTelegramHTML splits text so each chunk fits Telegram sendMessage limit.
func SplitTelegramHTML(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = maxTelegramMessageLen
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= maxLen {
		return []string{text}
	}
	var out []string
	lines := strings.Split(text, "\n")
	var buf strings.Builder
	for _, line := range lines {
		chunk := line
		if buf.Len() > 0 {
			chunk = "\n" + line
		}
		if buf.Len()+len(chunk) > maxLen {
			if buf.Len() > 0 {
				out = append(out, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
			for len(line) > maxLen {
				out = append(out, line[:maxLen])
				line = line[maxLen:]
			}
			if line != "" {
				buf.WriteString(line)
			}
			continue
		}
		buf.WriteString(chunk)
	}
	if buf.Len() > 0 {
		out = append(out, strings.TrimSpace(buf.String()))
	}
	return out
}

// SendHTMLMessageRetry sends HTML, splitting long text and retrying rate limits.
func (c *Client) SendHTMLMessageRetry(ctx context.Context, chatID int64, text string) error {
	if c == nil {
		return nil
	}
	parts := SplitTelegramHTML(text, maxTelegramMessageLen)
	if len(parts) == 0 {
		return nil
	}
	for i, part := range parts {
		if err := c.sendHTMLWithRetries(ctx, chatID, part, 4); err != nil {
			return err
		}
		if i+1 < len(parts) {
			time.Sleep(150 * time.Millisecond)
		}
	}
	return nil
}

func (c *Client) sendHTMLWithRetries(ctx context.Context, chatID int64, text string, attempts int) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		lastErr = c.SendHTMLMessage(ctx, chatID, text)
		if lastErr == nil {
			return nil
		}
		if sec := RetryAfterSec(lastErr); sec > 0 {
			timer := time.NewTimer(time.Duration(sec) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if attempt+1 < attempts {
			time.Sleep(time.Duration(attempt+1) * 400 * time.Millisecond)
		}
	}
	return lastErr
}
