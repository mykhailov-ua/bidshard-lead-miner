package warrior

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

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
	workers  int
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		var err error
		client, err = httpclient.NewClientWithProxies(cfg.HTTPTimeout, cfg.ProxyURLs)
		if err != nil {
			client = httpclient.Shared(cfg.HTTPTimeout)
		}
	}
	seedPath := cfg.WarriorSeedPath
	if seedPath == "" {
		seedPath = "data/seeds/warrior_threads.csv"
	}
	workers := cfg.HTTPWorkers
	if workers <= 0 {
		workers = 10
	}
	return &Crawler{
		seedPath: seedPath,
		client:   client,
		baseURL:  cfg.ForumBaseURL,
		workers:  workers,
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

	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(c.workers)

	for _, threadURL := range urls {
		tURL := threadURL
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			fetchURL := tURL
			if c.baseURL != "" {
				if u, err := url.Parse(tURL); err == nil {
					fetchURL = strings.TrimRight(c.baseURL, "/") + u.Path
				}
			}

			req, err := http.NewRequestWithContext(gCtx, http.MethodGet, fetchURL, nil)
			if err != nil {
				return err
			}

			resp, err := c.client.Do(req)
			if err != nil {
				slog.Warn("warrior fetch failed", "url", fetchURL, "error", err)
				return nil
			}

			body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			resp.Body.Close()
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				slog.Warn("warrior http status", "url", fetchURL, "status", resp.StatusCode)
				return nil
			}

			posts := parsePosts(string(body))
			source := sourceName(tURL)
			for _, post := range posts {
				contacts := extract.Extract(post.Body)
				if contacts.Rejected {
					continue
				}
				primary := ""
				if len(contacts.Contacts) > 0 {
					primary = extract.FormatAll(contacts.Contacts)[0]
				} else if post.Author != "" && post.Author != "anonymous" {
					primary = "warrior:user/" + post.Author
				} else {
					continue
				}
				item := model.RawItem{
					Source:   source,
					Raw:      post.Body,
					Contact:  primary,
					Title:    post.Author,
					PostedAt: post.PostedAt,
				}

				mu.Lock()
				if err := emit(gCtx, item); err != nil {
					mu.Unlock()
					return err
				}
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		slog.Warn("warrior crawl completed with error", "error", err)
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

var defaultStripper = NewFastHTMLStripper(65536)

func parsePosts(html string) []Post {
	matches := postContentRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	authors := authorRe.FindAllStringSubmatch(html, -1)
	times := postTimeRe.FindAllStringSubmatch(html, -1)

	var posts []Post
	for i, match := range matches {
		body := defaultStripper.StripTagsString(match[1])
		author := "anonymous"
		if i < len(authors) {
			author = defaultStripper.StripTagsString(authors[i][1])
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
	return defaultStripper.StripTagsString(s)
}

