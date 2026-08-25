package reviews

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Review struct {
	Author   string
	Rating   int
	Body     string
	Product  string
	PostedAt time.Time
}

type Crawler struct {
	client  *http.Client
	baseURL string
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		client = httpclient.CrawlClient(cfg.HTTPTimeout, cfg.ProxyURLsForSource("reviews"), "reviews")
	}
	return &Crawler{
		client:  client,
		baseURL: cfg.ForumBaseURL,
	}
}

func (c *Crawler) SetBaseURL(u string) {
	c.baseURL = u
}

func (c *Crawler) Name() string {
	return "reviews"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	products := []string{"voluum", "keitaro", "redtrack", "binom", "funnelflux"}
	for _, prod := range products {
		url := fmt.Sprintf("https://www.trustpilot.com/review/%s.com", prod)
		if c.baseURL != "" {
			url = c.baseURL + "/review/" + prod
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		body, status, err := httpclient.DoBytes(c.client, req, 2<<20)
		if err != nil {
			slog.Warn("reviews fetch failed", "product", prod, "error", err)
			continue
		}
		if status != http.StatusOK {
			continue
		}

		reviews := parseReviews(string(body), prod)
		for _, rev := range reviews {
			if rev.Rating > 2 {
				continue
			}
			contacts := extract.Extract(rev.Body)
			contactStr := ""
			if !contacts.Rejected && len(contacts.Contacts) > 0 {
				contactStr = extract.FormatAll(contacts.Contacts)[0]
			} else if rev.Author != "" {
				contactStr = "review:" + rev.Author
			}

			item := model.RawItem{
				Source:   fmt.Sprintf("reviews:trustpilot:%s", prod),
				Raw:      rev.Body,
				Contact:  contactStr,
				Title:    rev.Author,
				PostedAt: rev.PostedAt,
			}
			if err := emit(ctx, item); err != nil {
				return err
			}
		}
	}
	return nil
}

var (
	reviewCardRe = regexp.MustCompile(`(?is)<article[^>]*class="[^"]*review[^"]*"[^>]*>(.*?)</article>`)
	ratingRe     = regexp.MustCompile(`(?i)data-service-review-rating="(\d+)"`)
	reviewBodyRe = regexp.MustCompile(`(?is)<p[^>]*class="[^"]*review-text[^"]*"[^>]*>(.*?)</p>`)
	authorNameRe = regexp.MustCompile(`(?is)<span[^>]*class="[^"]*consumer-name[^"]*"[^>]*>(.*?)</span>`)
	reviewTimeRe = regexp.MustCompile(`(?is)<time[^>]*datetime="([^"]+)"`)
	stripTagRe   = regexp.MustCompile(`(?is)<[^>]+>`)
)

func parseReviews(html string, product string) []Review {
	cards := reviewCardRe.FindAllStringSubmatch(html, -1)
	if len(cards) == 0 {
		return nil
	}

	var out []Review
	for _, card := range cards {
		content := card[1]
		rating := 1
		if m := ratingRe.FindStringSubmatch(content); len(m) > 1 {
			if r, err := strconv.Atoi(m[1]); err == nil {
				rating = r
			}
		}
		body := ""
		if m := reviewBodyRe.FindStringSubmatch(content); len(m) > 1 {
			body = stripTags(m[1])
		}
		author := "anonymous"
		if m := authorNameRe.FindStringSubmatch(content); len(m) > 1 {
			author = stripTags(m[1])
		}

		if body == "" {
			continue
		}

		postedAt := time.Now().UTC()
		if m := reviewTimeRe.FindStringSubmatch(content); len(m) > 1 {
			if t, err := time.Parse(time.RFC3339, m[1]); err == nil {
				postedAt = t.UTC()
			}
		}

		out = append(out, Review{
			Author:   author,
			Rating:   rating,
			Body:     body,
			Product:  product,
			PostedAt: postedAt,
		})
	}
	return out
}

func stripTags(s string) string {
	s = stripTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
