package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestCrawlerCollect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":        "1",
				"content":   "voluum alternative postback failing",
				"timestamp": "2026-08-16T10:00:00Z",
				"author":    map[string]string{"username": "buyer_mx"},
			},
		})
	}))
	defer srv.Close()

	c := NewCrawler(config.Config{
		DiscordBotToken:   "test-token",
		DiscordChannelIDs: []string{"123"},
		DiscordMaxMessages: 10,
	})
	c.baseURL = srv.URL
	c.client = srv.Client()

	var items []model.RawItem
	err := c.Collect(context.Background(), func(_ context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].Contact != "discord:buyer_mx" {
		t.Fatalf("contact=%q", items[0].Contact)
	}
	if items[0].PostedAt.IsZero() {
		t.Fatal("expected posted_at")
	}
}
