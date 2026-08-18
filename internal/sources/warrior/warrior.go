package warrior

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	seedPath string
	client   *http.Client
	baseURL  string
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		client = httpclient.Shared(cfg.HTTPTimeout)
	}
	seedPath := cfg.WarriorSeedPath
	if seedPath == "" {
		seedPath = "data/seeds/warrior_threads.csv"
	}
	return &Crawler{
		seedPath: seedPath,
		client:   client,
		baseURL:  cfg.ForumBaseURL,
	}
}

func (c *Crawler) Name() string {
	return "warrior"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	urls, err := LoadThreadURLs(c.seedPath)
	if err != nil {
		return err
	}

	for _, threadURL := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fetchURL := threadURL
		if c.baseURL != "" {
			if u, err := url.Parse(threadURL); err == nil {
				fetchURL = strings.TrimRight(c.baseURL, "/") + u.Path
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
		if err != nil {
			return err
		}

		resp, err := c.client.Do(req)
		if err != nil {
			slog.Warn("warrior fetch failed", "url", fetchURL, "error", err)
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			slog.Warn("warrior http status", "url", fetchURL, "status", resp.StatusCode)
			continue
		}

		posts := parsePosts(string(body))
		source := sourceName(threadURL)
		for _, post := range posts {
			contacts := extract.Extract(post.Body)
			if contacts.Rejected || len(contacts.Contacts) == 0 {
				continue
			}
			primary := extract.FormatAll(contacts.Contacts)[0]
			item := model.RawItem{
				Source:   source,
				Raw:      post.Body,
				Contact:  primary,
				Title:    post.Author,
				PostedAt: post.PostedAt,
			}
			if err := emit(ctx, item); err != nil {
				return err
			}
		}
	}

	return nil
}

type Post struct {
	Author   string
	Body     string
	PostedAt time.Time
}

var (
	postContentRe = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*post-content[^"]*"[^>]*>(.*?)</div>`)
	authorRe      = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*username[^"]*"[^>]*>(.*?)</a>`)
	postTimeRe    = regexp.MustCompile(`(?is)<time[^>]*datetime="([^"]+)"`)
)

func parsePosts(html string) []Post {
	matches := postContentRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	authors := authorRe.FindAllStringSubmatch(html, -1)
	times := postTimeRe.FindAllStringSubmatch(html, -1)

	var posts []Post
	for i, match := range matches {
		body := stripTags(match[1])
		author := "anonymous"
		if i < len(authors) {
			author = stripTags(authors[i][1])
		}
		postedAt := time.Now().UTC()
		if i < len(times) {
			if t, err := time.Parse(time.RFC3339, times[i][1]); err == nil {
				postedAt = t.UTC()
			}
		}
		if body == "" {
			continue
		}
		posts = append(posts, Post{
			Author:   author,
			Body:     body,
			PostedAt: postedAt,
		})
	}
	return posts
}

func sourceName(threadURL string) string {
	u, err := url.Parse(threadURL)
	if err != nil {
		return "warrior:unknown"
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "threads" {
		return "warrior:" + parts[1]
	}
	return "warrior:" + strings.ReplaceAll(strings.Trim(u.Path, "/"), "/", "-")
}

func stripTags(s string) string {
	s = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
