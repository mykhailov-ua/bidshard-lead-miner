package reddit

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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{
					"id":       "abc123",
					"title":    "voluum alternative needed",
					"selftext": "postback failing on FTD every week",
					"author":   "media_buyer",
				},
			},
		})
	}))
	defer srv.Close()

	cfg := config.Config{
		RedditSubreddits: []string{"affiliatemarketing"},
		RedditQueries:    []string{"voluum alternative"},
		RedditMaxResults: 5,
	}
	c := NewCrawler(cfg)
	c.baseURL = srv.URL + "/?"
	c.client = srv.Client()

	var got []model.RawItem
	err := c.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		got = append(got, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("items=%d want 1", len(got))
	}
	if got[0].Contact != "reddit:u/media_buyer" {
		t.Fatalf("contact=%q", got[0].Contact)
	}
}
