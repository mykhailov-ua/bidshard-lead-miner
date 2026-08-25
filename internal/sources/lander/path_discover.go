package lander

import (
	"context"
	"encoding/xml"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

const (
	defaultLanderPathsCachePath = "data/runtime/lander_paths.json"
	maxSitemapFetches           = 2
	maxDiscoveredPaths          = 12
)

// PathFetcher loads robots.txt, sitemaps, and HTML for path discovery.
type PathFetcher interface {
	Get(ctx context.Context, rawURL string) (body string, status int, err error)
}

// PathRanker optionally narrows many candidate paths (Gemini).
type PathRanker interface {
	RankLanderPaths(ctx context.Context, domain string, candidates []string) ([]string, error)
}

// PathDiscoverConfig controls sitemap/robots/homepage path discovery.
type PathDiscoverConfig struct {
	Enabled   bool
	CachePath string
	Ranker    PathRanker
}

// PathDiscoverer discovers affiliate LPR paths beyond NetworkLandingPaths().
type PathDiscoverer struct {
	fetch PathFetcher
	cfg   PathDiscoverConfig
}

func NewPathDiscoverer(fetch PathFetcher, cfg PathDiscoverConfig) *PathDiscoverer {
	if cfg.CachePath == "" {
		cfg.CachePath = defaultLanderPathsCachePath
	}
	return &PathDiscoverer{fetch: fetch, cfg: cfg}
}

// PathsForDomain returns crawl paths for a domain (fixed list + discovered extras).
func (d *PathDiscoverer) PathsForDomain(ctx context.Context, domain string) []string {
	base := NetworkLandingPaths()
	if d == nil || !d.cfg.Enabled || d.fetch == nil {
		return base
	}
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return base
	}
	if cached, ok := loadLanderPathsCache(d.cfg.CachePath, domain); ok {
		return MergeLandingPaths(base, cached)
	}

	discovered := d.discover(ctx, domain)
	if len(discovered) > 0 {
		if err := saveLanderPathsCache(d.cfg.CachePath, domain, discovered); err != nil {
			slog.Debug("lander paths cache write failed", "domain", domain, "error", err)
		}
	}
	return MergeLandingPaths(base, discovered)
}

func (d *PathDiscoverer) discover(ctx context.Context, domain string) []string {
	var candidates []string
	for _, robotsURL := range []string{
		"https://" + domain + "/robots.txt",
		"https://www." + domain + "/robots.txt",
	} {
		body, status, err := d.fetch.Get(ctx, robotsURL)
		if err != nil || status != 200 {
			continue
		}
		for _, sm := range ParseRobotsSitemaps(body, robotsURL) {
			budget := maxSitemapFetches
			candidates = append(candidates, d.pathsFromSitemapURL(ctx, sm, &budget)...)
		}
		break
	}

	if homePaths := d.pathsFromHomepage(ctx, domain); len(homePaths) > 0 {
		candidates = append(candidates, homePaths...)
	}

	candidates = uniqueAffiliatePaths(candidates)
	if len(candidates) == 0 {
		return nil
	}

	if d.cfg.Ranker != nil && len(candidates) > maxDiscoveredPaths {
		ranked, err := d.cfg.Ranker.RankLanderPaths(ctx, domain, candidates)
		if err != nil {
			slog.Warn("lander path gemini rank failed", "domain", domain, "error", err)
		} else if len(ranked) > 0 {
			candidates = ranked
		}
	}

	if len(candidates) > maxDiscoveredPaths {
		candidates = candidates[:maxDiscoveredPaths]
	}
	slog.Info("lander paths discovered", "domain", domain, "paths", len(candidates))
	return candidates
}

func (d *PathDiscoverer) pathsFromSitemapURL(ctx context.Context, sitemapURL string, budget *int) []string {
	if budget != nil {
		if *budget <= 0 {
			return nil
		}
		*budget--
	}
	body, status, err := d.fetch.Get(ctx, sitemapURL)
	if err != nil || status != 200 {
		return nil
	}
	if child := ParseSitemapIndexLocs(body); len(child) > 0 {
		var out []string
		for _, loc := range child {
			out = append(out, d.pathsFromSitemapURL(ctx, loc, budget)...)
		}
		return out
	}
	var out []string
	for _, loc := range ParseSitemapPageLocs(body) {
		if path := URLToLandingPath(loc); path != "" && IsAffiliateLandingPath(path) {
			out = append(out, path)
		}
	}
	return out
}

func (d *PathDiscoverer) pathsFromHomepage(ctx context.Context, domain string) []string {
	for _, homeURL := range []string{"https://" + domain + "/", "https://www." + domain + "/"} {
		body, status, err := d.fetch.Get(ctx, homeURL)
		if err != nil || status != 200 {
			continue
		}
		return ExtractLinkPaths(body, domain)
	}
	return nil
}

var robotsSitemapRe = regexp.MustCompile(`(?im)^\s*sitemap:\s*(\S+)`)

// ParseRobotsSitemaps extracts Sitemap: URLs from robots.txt.
func ParseRobotsSitemaps(body, robotsURL string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, m := range robotsSitemapRe.FindAllStringSubmatch(body, -1) {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		if abs, err := resolveAgainst(raw, robotsURL); err == nil {
			raw = abs
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	return out
}

// ParseSitemapPageLocs returns page <loc> entries from a urlset sitemap.
func ParseSitemapPageLocs(xmlBody string) []string {
	type urlEntry struct {
		Loc string `xml:"loc"`
	}
	type urlSet struct {
		URLs []urlEntry `xml:"url"`
	}
	var set urlSet
	if err := xml.Unmarshal([]byte(xmlBody), &set); err != nil {
		return nil
	}
	out := make([]string, 0, len(set.URLs))
	for _, u := range set.URLs {
		if loc := strings.TrimSpace(u.Loc); loc != "" {
			out = append(out, loc)
		}
	}
	return out
}

// ParseSitemapIndexLocs returns child sitemap URLs from a sitemapindex.
func ParseSitemapIndexLocs(xmlBody string) []string {
	type siteMap struct {
		Loc string `xml:"loc"`
	}
	type siteMapIndex struct {
		Maps []siteMap `xml:"sitemap"`
	}
	var idx siteMapIndex
	if err := xml.Unmarshal([]byte(xmlBody), &idx); err != nil {
		return nil
	}
	out := make([]string, 0, len(idx.Maps))
	for _, sm := range idx.Maps {
		if loc := strings.TrimSpace(sm.Loc); loc != "" {
			out = append(out, loc)
		}
	}
	return out
}

// ParseSitemapLocs returns all <loc> entries from urlset or sitemapindex XML.
func ParseSitemapLocs(xmlBody string) []string {
	if pages := ParseSitemapPageLocs(xmlBody); len(pages) > 0 {
		return pages
	}
	return ParseSitemapIndexLocs(xmlBody)
}

var hrefRe = regexp.MustCompile(`(?i)href\s*=\s*["']([^"'#]+)["']`)

// ExtractLinkPaths pulls same-host anchor paths that look like affiliate LPR pages.
func ExtractLinkPaths(html, domain string) []string {
	domain = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "www.")
	var out []string
	seen := map[string]struct{}{}
	for _, m := range hrefRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		path := URLToLandingPath(strings.TrimSpace(m[1]))
		if path == "" || !IsAffiliateLandingPath(path) {
			continue
		}
		if !linkMatchesDomain(m[1], domain) {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func linkMatchesDomain(href, domain string) bool {
	href = strings.TrimSpace(href)
	if href == "" {
		return false
	}
	if strings.HasPrefix(href, "/") {
		return true
	}
	u, err := url.Parse(href)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	return host == "" || host == domain
}

// URLToLandingPath normalizes a URL or path to a site path starting with /.
func URLToLandingPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
			raw = raw[:idx]
		}
		return normalizePath(raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return normalizePath(u.Path)
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if idx := strings.Index(path, "//"); idx >= 0 {
		return ""
	}
	return path
}

var affiliatePathExclude = []string{
	"/blog", "/news", "/career", "/job", "/login", "/signin", "/signup", "/register",
	"/wp-", "/tag/", "/category/", "/feed", "/api/", "/assets/", "/static/",
}

var affiliatePathKeywords = []string{
	"affiliate", "partner", "contact", "about", "publisher", "advertiser",
	"become", "join", "cooperation", "mediabuy", "media-buy", "for-partners",
	"for-affiliates", "lpr", "partnership",
}

// IsAffiliateLandingPath reports whether a path is worth crawling for LPR contacts.
func IsAffiliateLandingPath(path string) bool {
	path = normalizePath(path)
	if path == "" {
		return false
	}
	lower := strings.ToLower(path)
	for _, ex := range affiliatePathExclude {
		if strings.Contains(lower, ex) {
			return false
		}
	}
	for _, kw := range affiliatePathKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// MergeLandingPaths appends discovered paths after the fixed network list (deduped).
func MergeLandingPaths(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	add := func(p string) {
		p = normalizePath(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range base {
		add(p)
	}
	for _, p := range extra {
		add(p)
	}
	return out
}

func uniqueAffiliatePaths(paths []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range paths {
		p = normalizePath(p)
		if p == "" || !IsAffiliateLandingPath(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func resolveAgainst(raw, base string) (string, error) {
	ref, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(ref).String(), nil
}

// HTTPPathFetcher adapts HTTPFetcher for path discovery.
type HTTPPathFetcher struct {
	HTTP *HTTPFetcher
}

func (f HTTPPathFetcher) Get(ctx context.Context, rawURL string) (string, int, error) {
	if f.HTTP == nil {
		return "", 0, context.Canceled
	}
	return f.HTTP.GetStatus(ctx, rawURL)
}
