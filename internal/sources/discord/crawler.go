package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
)

const apiBase = "https://discord.com/api/v10"

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	token      string
	channelIDs []string
	maxMsgs    int
	client     *http.Client
	baseURL    string
}

func NewCrawler(cfg config.Config) *Crawler {
	maxMsgs := cfg.DiscordMaxMessages
	if maxMsgs <= 0 {
		maxMsgs = 50
	}
	return &Crawler{
		token:      strings.TrimSpace(cfg.DiscordBotToken),
		channelIDs: cfg.DiscordChannelIDs,
		maxMsgs:    maxMsgs,
		client:     httpclient.Shared(cfg.HTTPTimeout),
		baseURL:    apiBase,
	}
}

func (c *Crawler) Name() string {
	return "discord"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	if c.token == "" || len(c.channelIDs) == 0 {
		slog.Warn("discord crawl skipped", "reason", "missing token or channel ids")
		return nil
	}

	start := time.Now()
	emitted := 0
	seen := map[string]struct{}{}

	for _, channelID := range c.channelIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := c.fetchMessages(ctx, channelID)
		if err != nil {
			slog.Warn("discord channel fetch failed", "channel_id", channelID, "error", err)
			continue
		}
		for _, msg := range msgs {
			if msg.ID != "" {
				if _, ok := seen[msg.ID]; ok {
					continue
				}
				seen[msg.ID] = struct{}{}
			}
			text := strings.TrimSpace(msg.Content)
			if text == "" {
				continue
			}
			author := strings.TrimSpace(msg.Author.Username)
			if author == "" {
				continue
			}
			postedAt := time.Now().UTC()
			if msg.Timestamp != "" {
				if t, err := time.Parse(time.RFC3339Nano, msg.Timestamp); err == nil {
					postedAt = t.UTC()
				} else if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
					postedAt = t.UTC()
				}
			}

			item := model.RawItem{
				Source:   "discord:" + channelID,
				Raw:      text,
				Title:    truncate(text, 120),
				Contact:  "discord:" + author,
				PostedAt: postedAt,
			}
			if err := emit(ctx, item); err != nil {
				return err
			}
			emitted++
		}
	}

	slog.Info("discord crawl finished",
		"channels", len(c.channelIDs),
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

type message struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Author    struct {
		Username string `json:"username"`
	} `json:"author"`
}

func (c *Crawler) fetchMessages(ctx context.Context, channelID string) ([]message, error) {
	url := fmt.Sprintf("%s/channels/%s/messages?limit=%d", c.baseURL, channelID, c.maxMsgs)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bot "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord http %d", resp.StatusCode)
	}

	var msgs []message
	if err := json.Unmarshal(body, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
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
