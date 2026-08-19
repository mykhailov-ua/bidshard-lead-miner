package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type SearchResponse struct {
	Items []IssueItem `json:"items"`
}

type IssueItem struct {
	HTMLURL   string `json:"html_url"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	User      User   `json:"user"`
}

type User struct {
	Login string `json:"login"`
}

type Crawler struct {
	token   string
	queries []string
	client  *http.Client
	baseURL string
}

func NewCrawler(cfg config.Config) *Crawler {
	queries := cfg.GitHubSearchQueries
	if len(queries) == 0 {
		queries = []string{
			"tracker alternative",
			"openrtb",
			"clickhouse tracker",
			"voluum api",
			"keitaro api",
			"self-hosted tracker",
		}
	}
	client, err := httpclient.NewClientWithProxies(cfg.HTTPTimeout, cfg.ProxyURLs)
	if err != nil {
		client = httpclient.Shared(cfg.HTTPTimeout)
	}
	return &Crawler{
		token:   cfg.GitHubToken,
		queries: queries,
		client:  client,
		baseURL: "https://api.github.com",
	}
}

func (c *Crawler) SetBaseURL(u string) {
	c.baseURL = u
}

func (c *Crawler) Name() string {
	return "github"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	for _, q := range c.queries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		reqURL := fmt.Sprintf("%s/search/issues?q=%s+type:issue", c.baseURL, url.QueryEscape(q))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			slog.Warn("github search failed", "query", q, "error", err)
			continue
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == 429 {
			retryAfter := resp.Header.Get("Retry-After")
			slog.Warn("github rate limit reached", "query", q, "retry_after", retryAfter)
			resp.Body.Close()
			if sec, err := strconv.Atoi(retryAfter); err == nil && sec > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(sec) * time.Second):
				}
			}
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var searchRes SearchResponse
		if err := json.Unmarshal(body, &searchRes); err != nil {
			slog.Warn("github json parse failed", "query", q, "error", err)
			continue
		}

		for _, item := range searchRes.Items {
			combined := item.Title + " " + item.Body
			contacts := extract.Extract(combined)
			contactStr := ""
			if !contacts.Rejected && len(contacts.Contacts) > 0 {
				contactStr = extract.FormatAll(contacts.Contacts)[0]
			} else if item.User.Login != "" {
				contactStr = "github:" + item.User.Login
			}

			postedAt := time.Now().UTC()
			if item.CreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, item.CreatedAt); err == nil {
					postedAt = t.UTC()
				}
			}

			rawItem := model.RawItem{
				Source:   "github:" + parseRepoSlug(item.HTMLURL),
				Raw:      combined,
				Contact:  contactStr,
				Title:    item.Title,
				PostedAt: postedAt,
			}
			if err := emit(ctx, rawItem); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseRepoSlug(rawURL string) string {
	rawURL = strings.TrimPrefix(rawURL, "https://github.com/")
	parts := strings.Split(rawURL, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return rawURL
}
