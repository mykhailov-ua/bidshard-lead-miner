package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
)

const pullPushBase = "https://api.pullpush.io/reddit/search/submission/"

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	subreddits []string
	queries    []string
	maxResults int
	client     *http.Client
	baseURL    string
}

func NewCrawler(cfg config.Config) *Crawler {
	subs := cfg.RedditSubreddits
	if len(subs) == 0 {
		subs = []string{"affiliatemarketing", "media_buying", "juststart"}
	}
	queries := cfg.RedditQueries
	if len(queries) == 0 {
		queries = []string{
			"voluum alternative",
			"tracker too expensive",
			"postback failing",
			"self-hosted tracker",
		}
	}
	maxResults := cfg.RedditMaxResults
	if maxResults <= 0 {
		maxResults = 25
	}
	return &Crawler{
		subreddits: subs,
		queries:    queries,
		maxResults: maxResults,
		client:     httpclient.Shared(cfg.HTTPTimeout),
		baseURL:    pullPushBase,
	}
}

func (c *Crawler) Name() string {
	return "reddit"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	start := time.Now()
	emitted := 0
	seen := map[string]struct{}{}

	for _, sub := range c.subreddits {
		for _, query := range c.queries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			items, err := c.search(ctx, sub, query)
			if err != nil {
				slog.Warn("reddit search failed", "subreddit", sub, "query", query, "error", err)
				continue
			}
			for _, post := range items {
				key := post.ID
				if key == "" {
					key = post.Permalink
				}
				if key == "" {
					continue
				}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				text := strings.TrimSpace(strings.Join([]string{post.Title, post.SelfText}, "\n"))
				if text == "" {
					continue
				}
				author := strings.TrimSpace(post.Author)
				if author == "" || strings.EqualFold(author, "[deleted]") {
					continue
				}

				item := model.RawItem{
					Source:  "reddit:r/" + sub,
					Raw:     text,
					Title:   post.Title,
					Contact: "reddit:u/" + author,
				}
				if post.CreatedUTC > 0 {
					item.PostedAt = time.Unix(int64(post.CreatedUTC), 0).UTC()
				}
				if err := emit(ctx, item); err != nil {
					return err
				}
				emitted++
			}
		}
	}

	slog.Info("reddit crawl finished",
		"subreddits", len(c.subreddits),
		"queries", len(c.queries),
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

type submission struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	SelfText   string  `json:"selftext"`
	Author     string  `json:"author"`
	Permalink  string  `json:"permalink"`
	CreatedUTC float64 `json:"created_utc"`
}

type searchResponse struct {
	Data []submission `json:"data"`
}

func (c *Crawler) search(ctx context.Context, subreddit, query string) ([]submission, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("subreddit", subreddit)
	params.Set("size", fmt.Sprintf("%d", c.maxResults))
	params.Set("sort", "desc")
	params.Set("sort_type", "created_utc")

	reqURL := c.baseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("pullpush http %d", resp.StatusCode)
	}

	var parsed searchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	return parsed.Data, nil
}
