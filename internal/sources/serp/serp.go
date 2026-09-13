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
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	client            *http.Client
	dorks             []string
	maxResults        int
	telegramDorkMax   int
	dorkOffset        int
	dorkBatch         int
	baseURL           string
	disabledDorksPath string
}

func NewCrawler(cfg config.Config, client *http.Client) *Crawler {
	if client == nil {
		client = httpclient.CrawlClient(cfg.HTTPTimeout, cfg.ProxyURLsForSource("serp"), "serp")
	}
	dorks := []string{
		`site:blackhatworld.com "voluum alternative"`,
		`site:affiliatefix.com "tracker too expensive"`,
		`site:warriorforum.com "self-hosted tracker"`,
		`"voluum too expensive" affiliate`,
		`"keitaro alternative" tracker`,
		`site:t.me affiliate marketing`,
		`site:t.me igaming affiliate`,
		`site:t.me media buying team`,
		`site:t.me usdt tracker`,
		`site:tgstat.com affiliate`,
		`telegram channel affiliate marketing tracker`,
	}
	path := cfg.DisabledDorksPath
	if path == "" {
		path = dorkdisable.DefaultPath
	}
	return &Crawler{
		client:            client,
		dorks:             dorkdisable.FilterActiveDorks(path, dorks),
		maxResults:        20,
		telegramDorkMax:   cfg.SerpTelegramDorkMax,
		dorkOffset:        cfg.SerpDorkOffset,
		dorkBatch:         cfg.SerpDorkBatch,
		baseURL:           "https://html.duckduckgo.com/html/",
		disabledDorksPath: path,
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

		results, err := c.searchDork(ctx, dork)
		if err != nil {
			slog.Warn("serp fetch failed", "dork", dork, "error", err)
			continue
		}
		if err := appendTelegramChannelDiscoveries(defaultTGChannelsPath, dork, results); err != nil {
			slog.Warn("serp telegram channel registry write failed", "error", err)
		}
		forumItems := ExtractForumThreadDiscoveries(results)
		if len(forumItems) > 0 {
			added, err := forum.AppendThreadDiscoveries(defaultForumThreadsPath, "serp_poll", dork, forumItems)
			if err != nil {
				slog.Warn("serp forum thread registry write failed", "error", err)
			} else if added > 0 {
				slog.Debug("serp forum threads queued", "dork", dork, "added", added)
			}
		}
		for _, res := range results {
			if forum.IsKnownForumHost(res.Domain) || forum.IsForumThreadURL(res.URL) {
				continue
			}
			contacts := extract.Extract(res.Snippet)
			contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
			if contacts.Rejected || len(contacts.Contacts) == 0 {
				continue
			}
			contactStr := extract.FormatAll(contacts.Contacts)[0]

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
	return geo.IsBlockedTLD(domain)
}

func stripTags(s string) string {
	s = stripTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
