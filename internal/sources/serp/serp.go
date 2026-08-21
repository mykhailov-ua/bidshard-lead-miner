package serp

import (
	"context"
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
	client     *http.Client
	dorks      []string
	maxResults int
	baseURL    string
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		client = httpclient.CrawlClient(cfg.HTTPTimeout, cfg.ProxyURLs)
	}
	dorks := []string{
		`site:blackhatworld.com "voluum alternative"`,
		`site:affiliatefix.com "tracker too expensive"`,
		`site:warriorforum.com "self-hosted tracker"`,
		`"voluum too expensive" affiliate`,
		`"keitaro alternative" tracker`,
		`site:t.me affiliate marketing`,
		`site:t.me igaming affiliate`,
		`site:tgstat.com affiliate`,
		`telegram channel affiliate marketing tracker`,
	}
	return &Crawler{
		client:     client,
		dorks:      dorks,
		maxResults: 20,
		baseURL:    "https://html.duckduckgo.com/html/",
	}
}

func (c *Crawler) SetBaseURL(u string) {
	c.baseURL = u
}

func (c *Crawler) Name() string {
	return "serp"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	for _, dork := range c.dorks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		params := url.Values{}
		params.Set("q", dork)
		reqURL := c.baseURL
		if !strings.HasSuffix(reqURL, "/") {
			reqURL += "/"
		}
		if !strings.Contains(reqURL, "?") {
			reqURL += "?" + params.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			slog.Warn("serp request build failed", "dork", dork, "error", err)
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		body, status, err := httpclient.DoBytes(c.client, req, 2<<20)
		if err != nil {
			slog.Warn("serp fetch failed", "dork", dork, "error", err)
			continue
		}
		if status != http.StatusOK {
			slog.Warn("serp bad status", "dork", dork, "status", status)
			continue
		}

		results := parseSERPResults(string(body))
		if err := appendTelegramChannelDiscoveries(defaultTGChannelsPath, dork, results); err != nil {
			slog.Warn("serp telegram channel registry write failed", "error", err)
		}
		for _, res := range results {
			contacts := extract.Extract(res.Snippet)
			contactStr := ""
			if !contacts.Rejected && len(contacts.Contacts) > 0 {
				contactStr = extract.FormatAll(contacts.Contacts)[0]
			} else {
				contactStr = "serp:" + res.Domain
			}

			item := model.RawItem{
				Source:   "serp:" + res.Domain,
				Raw:      res.Title + " - " + res.Snippet,
				Contact:  contactStr,
				Title:    res.Title,
				PostedAt: time.Now().UTC(),
			}
			if err := emit(ctx, item); err != nil {
				return err
			}
		}
	}
	return nil
}

type SERPResult struct {
	Title   string
	Snippet string
	URL     string
	Domain  string
}

var (
	linkRe     = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	snippetRe  = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>`)
	stripTagRe = regexp.MustCompile(`(?is)<[^>]+>`)
)

func parseSERPResults(html string) []SERPResult {
	links := linkRe.FindAllStringSubmatch(html, -1)
	if len(links) == 0 {
		return nil
	}
	snippets := snippetRe.FindAllStringSubmatch(html, -1)

	var results []SERPResult
	for i, match := range links {
		rawURL := match[1]
		title := stripTags(match[2])
		snippet := ""
		if i < len(snippets) {
			snippet = stripTags(snippets[i][1])
		}

		domain := extractDomain(rawURL)
		if domain == "" || isBlockedSERPDomain(domain) {
			continue
		}

		results = append(results, SERPResult{
			Title:   title,
			Snippet: snippet,
			URL:     rawURL,
			Domain:  domain,
		})
	}
	return results
}

func extractDomain(rawURL string) string {
	rawURL = strings.TrimPrefix(rawURL, "//duckduckgo.com/l/?uddg=")
	if idx := strings.Index(rawURL, "&"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if decoded, err := url.QueryUnescape(rawURL); err == nil {
		rawURL = decoded
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

func isBlockedSERPDomain(domain string) bool {
	return strings.HasSuffix(domain, ".ru") || strings.HasSuffix(domain, ".by") || strings.HasSuffix(domain, ".su")
}

func stripTags(s string) string {
	s = stripTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
