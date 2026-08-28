package supply

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/domaincascade"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/validate"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	seedPath      string
	registryPath  string
	tgDomainsPath string
	domainTriage  bool
	maxHosts      int
	usesProxy     bool
	fetcher       *Fetcher
}

func NewCrawler(cfg config.Config, fetcher *Fetcher) *Crawler {
	if fetcher == nil {
		fetcher = NewFetcherFromConfig(cfg)
	}
	return &Crawler{
		seedPath:      cfg.SupplySeedPath,
		registryPath:  cfg.SourceRegistryPath,
		tgDomainsPath: cfg.TelegramDomainsPath,
		domainTriage:  cfg.ParserDomainTriage,
		maxHosts:      cfg.SupplyMaxDomains,
		usesProxy:     len(cfg.ProxyURLsForSource("supply")) > 0,
		fetcher:       fetcher,
	}
}

func (c *Crawler) Name() string {
	return "supply"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("supply", c.usesProxy); skip {
		slog.Info("supply crawl skipped", "reason", reason)
		return nil
	}
	domains, err := LoadSeedDomainsCombined(c.seedPath, c.registryPath)
	if err != nil {
		return err
	}
	if c.maxHosts > 0 && len(domains) > c.maxHosts {
		domains = domains[:c.maxHosts]
	}

	start := time.Now()
	emitted := 0
	skippedTriage := 0

	for _, domain := range domains {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		meta := sourceregistry.DomainMeta{Domain: domain}
		if skip, reason := sourceregistry.ShouldSkipCrawl(c.registryPath, c.domainTriage, meta); skip {
			skippedTriage++
			if strings.HasPrefix(reason, "heuristic:") {
				metrics.RecordSourcesTriagedDropped(1)
				_ = sourceregistry.SetTriageStatus(c.registryPath, domain, "drop")
			}
			slog.Debug("supply domain skipped triage", "domain", domain, "reason", reason)
			continue
		}

		count, err := c.crawlDomain(ctx, domain, emit)
		if err != nil {
			slog.Warn("supply domain crawl failed", "domain", domain, "error", err)
			continue
		}
		emitted += count
	}

	slog.Info("supply crawl finished",
		"domains", len(domains),
		"emitted", emitted,
		"skipped_triage", skippedTriage,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func (c *Crawler) crawlDomain(ctx context.Context, domain string, emit EmitFunc) (int, error) {
	emitted := 0

	adsBody, adsCode, adsErr := c.fetcher.Get(ctx, domain, "/ads.txt")
	var adsLines []AdsTxtLine
	if adsErr == nil {
		adsLines = ParseAdsTxt(string(adsBody))
	} else if adsCode != http.StatusNotFound {
		slog.Debug("ads.txt fetch", "domain", domain, "error", adsErr)
	}

	appAdsBody, _, appAdsErr := c.fetcher.Get(ctx, domain, "/app-ads.txt")
	if appAdsErr == nil {
		appLines := ParseAdsTxt(string(appAdsBody))
		adsLines = append(adsLines, appLines...)
	}

	sellersBody, sellersCode, sellersErr := c.fetcher.Get(ctx, domain, "/sellers.json")
	var sellers []SellerContact
	if sellersErr == nil {
		sellers = ParseSellersJSON(sellersBody)
	} else if sellersCode != http.StatusNotFound {
		slog.Debug("sellers.json fetch", "domain", domain, "error", sellersErr)
	}

	if len(adsLines) == 0 && len(sellers) == 0 {
		return 0, nil
	}

	partners := PartnerDomainsFromCrawl(domain, adsLines, sellers)
	if len(partners) > 0 {
		_, _, cascadeErr := domaincascade.FanOutMany(domaincascade.Config{
			RegistryPath:        c.registryPath,
			TelegramDomainsPath: c.tgDomainsPath,
		}, partners, "ads_txt", "supply")
		if cascadeErr != nil {
			slog.Debug("supply cascade failed", "domain", domain, "error", cascadeErr)
		}
	}

	snippet := BuildSnippet(domain, adsLines, sellers, string(adsBody), string(appAdsBody))
	crawlHTML := string(adsBody) + "\n" + string(appAdsBody) + "\n" + string(sellersBody)
	contacts := collectContacts(sellers)
	if directContact := ExtractContactDirective(string(adsBody)); directContact != "" && validate.AcceptEmail(directContact) {
		contacts = appendUniqueContact(contacts, directContact)
	}
	if directContact := ExtractContactDirective(string(appAdsBody)); directContact != "" && validate.AcceptEmail(directContact) {
		contacts = appendUniqueContact(contacts, directContact)
	}
	if len(contacts) == 0 {
		// Registry intel when sellers.json has no outreach-grade mailbox.
		contacts = append(contacts, "domain:"+domain)
	}

	for _, contact := range contacts {
		if res := geo.Filter("", contact); !res.OK {
			continue
		}
		item := model.RawItem{
			Source:    "ads_txt:" + domain,
			Raw:       snippet,
			Contact:   contact,
			Title:     "Supply contact",
			CrawlHTML: model.LimitCrawlHTML(crawlHTML),
		}
		if err := emit(ctx, item); err != nil {
			return emitted, err
		}
		emitted++
	}
	return emitted, nil
}

// collectContacts returns deduped seller emails that pass validate.AcceptEmail.
func collectContacts(sellers []SellerContact) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range sellers {
		email := strings.ToLower(strings.TrimSpace(s.ContactEmail))
		if email == "" || !validate.AcceptEmail(email) {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, email)
	}
	return out
}

func appendUniqueContact(contacts []string, email string) []string {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return contacts
	}
	for _, existing := range contacts {
		if existing == email {
			return contacts
		}
	}
	return append(contacts, email)
}
