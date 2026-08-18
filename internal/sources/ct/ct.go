package ct

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

type EmitFunc func(ctx context.Context, item model.RawItem) error

type CRTEntry struct {
	NameValue string `json:"name_value"`
}

type Crawler struct {
	queries    []string
	maxResults int
	client     *http.Client
	baseURL    string
	rateTicker *time.Ticker
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		client = httpclient.Shared(cfg.HTTPTimeout)
	}
	queries := cfg.CTQueries
	if len(queries) == 0 {
		queries = []string{"track", "click", "go"}
	}
	maxResults := cfg.CTMaxResults
	if maxResults <= 0 {
		maxResults = 100
	}
	return &Crawler{
		queries:    queries,
		maxResults: maxResults,
		client:     client,
		baseURL:    "https://crt.sh",
		rateTicker: time.NewTicker(2 * time.Second),
	}
}

func (c *Crawler) SetBaseURL(u string) {
	c.baseURL = u
}

func (c *Crawler) Name() string {
	return "ct"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	defer c.rateTicker.Stop()
	seen := make(map[string]struct{})

	for _, q := range c.queries {
		select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.rateTicker.C:
		}

		reqURL := fmt.Sprintf("%s/?q=%%25.%s.%%25&output=json", c.baseURL, url.QueryEscape(q))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			slog.Warn("ct request build failed", "query", q, "error", err)
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			slog.Warn("ct fetch failed", "query", q, "error", err)
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			slog.Warn("ct bad response", "query", q, "status", resp.StatusCode)
			continue
		}

		var entries []CRTEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			slog.Warn("ct json parse failed", "query", q, "error", err)
			continue
		}

		count := 0
		for _, entry := range entries {
			domains := strings.Split(entry.NameValue, "\n")
			for _, d := range domains {
				d = strings.ToLower(strings.TrimSpace(d))
				d = strings.TrimPrefix(d, "*.")
				if d == "" || isBlockedTLD(d) {
					continue
				}
				if _, exists := seen[d]; exists {
					continue
				}
				seen[d] = struct{}{}

				item := model.RawItem{
					Source:  "ct:" + d,
					Raw:     "Domain certificate transparency discovery: " + d,
					Contact: "domain:" + d,
					Title:   d,
				}
				if err := emit(ctx, item); err != nil {
					return err
				}
				count++
				if count >= c.maxResults {
					break
				}
			}
			if count >= c.maxResults {
				break
			}
		}
	}
	return nil
}

func isBlockedTLD(domain string) bool {
	return strings.HasSuffix(domain, ".ru") || strings.HasSuffix(domain, ".by") || strings.HasSuffix(domain, ".su")
}
