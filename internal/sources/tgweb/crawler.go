package tgweb

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources/lander"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	domainsPath  string
	registryPath string
	domainTriage bool
	maxDomains   int
	rescanDays   int
	domainFilter []string // non-empty = allowlist mode; ignores crawled_at
	usesProxy    bool
	pathDiscover lander.PathDiscoverConfig
	fetch        *lander.PageFetcher
}

// NewCrawler wires HTTP via NewHTTPFetcherFromConfig when httpFetcher is nil,
// so PARSER_PROXY_LIST applies to tgweb without changing global httpclient.Shared.
// ranker is optional Gemini path ranking when PARSER_LANDER_PATH_GEMINI=true.
func NewCrawler(cfg config.Config, httpFetcher *lander.HTTPFetcher, headless lander.HeadlessFetcher, ranker lander.PathRanker) (*Crawler, error) {
	if httpFetcher == nil {
		var err error
		httpFetcher, err = lander.NewHTTPFetcherForSource(cfg, "tgweb")
		if err != nil {
			return nil, err
		}
	}
	if headless == nil {
		headless = lander.DisabledHeadless{}
	}
	return &Crawler{
		domainsPath:  cfg.TelegramDomainsPath,
		registryPath: cfg.SourceRegistryPath,
		domainTriage: cfg.ParserDomainTriage,
		maxDomains:   cfg.TelegramWebMaxDomains,
		rescanDays:   cfg.TelegramWebRescanDays,
		domainFilter: append([]string(nil), cfg.TelegramWebDomains...),
		usesProxy:    len(cfg.ProxyURLsForSource("tgweb")) > 0,
		pathDiscover: lander.PathDiscoverConfig{
			Enabled:   cfg.ParserLanderPathDiscover,
			CachePath: cfg.LanderPathsCachePath,
			Ranker:    ranker,
		},
		fetch: lander.NewPageFetcher(httpFetcher, headless, lander.PageFetchOptsFromConfig(cfg, "tgweb")),
	}, nil
}

func (c *Crawler) Name() string {
	return "tgweb"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("tgweb", c.usesProxy); skip {
		slog.Info("tgweb crawl skipped", "reason", reason)
		return nil
	}
	file, err := LoadDomains(c.domainsPath)
	if err != nil {
		return err
	}
	pending := SelectDomains(file, c.maxDomains, c.rescanDays, c.domainFilter)
	if len(pending) == 0 {
		slog.Info("tgweb crawl skipped", "reason", "no pending domains")
		return nil
	}

	start := time.Now()
	var stats crawlRunStats
	skippedInvalid := 0
	skippedTriage := 0

	slog.Info("tgweb crawl started", "domains", len(pending), "path", c.domainsPath, "only", c.domainFilter)

	for _, entry := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		meta := sourceregistry.DomainMeta{
			Domain:       entry.Domain,
			Channel:      entry.Channel,
			Source:       entry.Source,
			DiscoveredBy: entry.Source,
			Kind:         entry.Kind,
		}
		if skip, reason := sourceregistry.ShouldSkipCrawl(c.registryPath, c.domainTriage, meta); skip {
			skippedTriage++
			if strings.HasPrefix(reason, "heuristic:") {
				metrics.RecordSourcesTriagedDropped(1)
				_ = sourceregistry.SetTriageStatus(c.registryPath, entry.Domain, "drop")
			}
			slog.Debug("tgweb domain skipped triage", "domain", entry.Domain, "reason", reason)
			continue
		}

		if !IsValidCrawlDomain(entry.Domain) {
			skippedInvalid++
			slog.Debug("tgweb domain skipped invalid", "domain", entry.Domain)
			// Drop invalid hosts from the retry queue; do not leave them pending forever.
			if markErr := MarkCrawled(c.domainsPath, entry.Domain); markErr != nil {
				slog.Warn("tgweb mark crawled failed", "domain", entry.Domain, "error", markErr)
			}
			continue
		}

		count, outcome := c.crawlDomain(ctx, entry, emit, &stats)
		switch outcome {
		case crawlOutcomeEmitted:
			// Only successful emits advance crawled_at; failures stay eligible for rescan.
			if markErr := MarkCrawled(c.domainsPath, entry.Domain); markErr != nil {
				slog.Warn("tgweb mark crawled failed", "domain", entry.Domain, "error", markErr)
			}
			stats.emitted += count
		case crawlOutcomeHardFail:
			stats.hardFail++
		case crawlOutcomeNoContacts:
			stats.noContacts++
		}
	}

	slog.Info("tgweb crawl finished",
		"domains", len(pending),
		"emitted", stats.emitted,
		"contacts_found", stats.contactsFound,
		"hard_fail", stats.hardFail,
		"no_contacts", stats.noContacts,
		"spa404_stopped", stats.spa404Stop,
		"skipped_invalid", skippedInvalid,
		"skipped_triage", skippedTriage,
		"deferred_retry", stats.hardFail+stats.noContacts,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

// crawlDomain walks NetworkLandingPaths until one page yields a site LPR contact.
// Stop early on SPA soft-404 shells to avoid downloading the same HTML for every path.
func (c *Crawler) crawlDomain(ctx context.Context, entry DomainEntry, emit EmitFunc, stats *crawlRunStats) (int, domainCrawlOutcome) {
	domain := entry.Domain
	paths := c.landingPaths(ctx, domain)
	emitted := 0

	var spa404FP string
	var spa404Logged bool

	for _, path := range paths {
		pageURL := "https://" + domain + path
		html, fetchMeta, status, err := c.fetchPage(ctx, pageURL, false)
		if err != nil {
			// Retry apex fetch with www prefix once; do not repeat for every affiliate path.
			if path == "/" {
				pageURL = "https://www." + domain + path
				html, fetchMeta, status, err = c.fetchPage(ctx, pageURL, false)
			}
			if err != nil {
				if lander.IsSPA404Shell(status, html) {
					fp := lander.SPAShellFingerprint(html)
					// Same shell fingerprint on a later path means the sweep will not surface new content.
					if spa404FP != "" && fp == spa404FP {
						if stats != nil {
							stats.spa404Stop++
						}
						slog.Debug("tgweb spa404 path sweep stopped",
							"domain", domain,
							"path", path,
							"fingerprint", fp,
						)
						break
					}
					if spa404FP == "" {
						spa404FP = fp
						if !spa404Logged {
							slog.Warn("lander http non-ok",
								"url", pageURL,
								"rsc", false,
								"status", status,
								"body_bytes", len(html),
								"body_preview", diag.Preview(html, 300),
								"hint", "spa404_shell",
							)
							spa404Logged = true
						}
						if path == "/" {
							if stats != nil {
								stats.spa404Stop++
							}
							slog.Debug("tgweb spa404 root shell skip paths",
								"domain", domain,
								"fingerprint", fp,
							)
							// Root already serves the catch-all shell; skip deeper paths.
							break
						}
					}
					continue
				}
				if path != "/" {
					// Non-root path errors are expected on sites without that route; keep sweeping.
					continue
				}
				slog.Warn("tgweb domain crawl hard fail", "domain", domain, "hint", "will retry on next run")
				return emitted, crawlOutcomeHardFail
			}
		}

		slog.Debug("tgweb fetch ok",
			"domain", domain,
			"path", path,
			"stage", fetchMeta.Stage,
			"rsc_fetched", fetchMeta.RSCFetched,
			"html_bytes", len(html),
		)

		text, method := lander.TextForContactExtract(html)
		if text == "" {
			continue
		}

		pageContacts := extract.Extract(text)
		if pageContacts.Rejected {
			continue
		}
		primary, ok := pickSiteLPR(pageContacts, entry.Channel, domain)
		if !ok {
			continue
		}

		source := sourceLabel(entry.Channel, domain)
		contact := formatPrimaryContact(primary)

		if stats != nil {
			stats.contactsFound++
		}

		slog.Info("tgweb contacts found",
			"domain", domain,
			"path", path,
			"method", method,
			"fetch_stage", fetchMeta.Stage,
			"channel", entry.Channel,
			"contact_type", primary.Type,
			"contact_preview", diag.MaskContact(contact),
		)

		item := model.RawItem{
			Source:    source,
			Raw:       text,
			Contact:   contact,
			Title:     provenanceTitle(entry.Channel, domain),
			CrawlHTML: model.LimitCrawlHTML(html),
		}
		if err := emit(ctx, item); err != nil {
			slog.Warn("tgweb emit failed", "domain", domain, "error", err)
			return emitted, crawlOutcomeHardFail
		}
		emitted++
		// Emit one lead per domain; first path with LPR wins.
		return emitted, crawlOutcomeEmitted
	}

	slog.Debug("tgweb domain no contacts", "domain", domain, "hint", "will retry on next run")
	return emitted, crawlOutcomeNoContacts
}

func (c *Crawler) landingPaths(ctx context.Context, domain string) []string {
	if c == nil || !c.pathDiscover.Enabled || c.fetch == nil || c.fetch.HTTP == nil {
		return lander.NetworkLandingPaths()
	}
	d := lander.NewPathDiscoverer(lander.HTTPPathFetcher{HTTP: c.fetch.HTTP}, c.pathDiscover)
	return d.PathsForDomain(ctx, domain)
}

func (c *Crawler) fetchPage(ctx context.Context, pageURL string, logHTTPNonOK bool) (string, lander.PageFetchMeta, int, error) {
	// Pass logHTTPNonOK=false on path sweeps; SPA404 detection consumes non-OK bodies without log spam.
	return c.fetch.FetchForCrawl(ctx, pageURL, logHTTPNonOK)
}

func sourceLabel(channel, domain string) string {
	channel = strings.TrimSpace(strings.TrimPrefix(channel, "@"))
	if channel != "" {
		return "tgweb:@" + strings.ToLower(channel) + ":" + domain
	}
	return "tgweb:" + domain
}
