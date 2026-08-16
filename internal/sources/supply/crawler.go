package supply

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	seedPath string
	maxHosts int
	fetcher  *Fetcher
}

func NewCrawler(cfg config.Config, fetcher *Fetcher) *Crawler {
	if fetcher == nil {
		fetcher = NewFetcher(cfg.HTTPTimeout, cfg.SupplyHostRPS, cfg.SupplyBaseURL)
	}
	return &Crawler{
		seedPath: cfg.SupplySeedPath,
		maxHosts: cfg.SupplyMaxDomains,
		fetcher:  fetcher,
	}
}

func (c *Crawler) Name() string {
	return "supply"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	domains, err := LoadSeedDomains(c.seedPath)
	if err != nil {
		return err
	}
	if c.maxHosts > 0 && len(domains) > c.maxHosts {
		domains = domains[:c.maxHosts]
	}

	start := time.Now()
	emitted := 0

	for _, domain := range domains {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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

	snippet := BuildSnippet(domain, adsLines, sellers)
	contacts := collectContacts(sellers)
	if len(contacts) == 0 {
		return 0, nil
	}

	for _, contact := range contacts {
		if res := geo.Filter("", contact); !res.OK {
			continue
		}
		item := model.RawItem{
			Source:  "ads_txt:" + domain,
			Raw:     snippet,
			Contact: contact,
			Title:   "Supply contact",
		}
		if err := emit(ctx, item); err != nil {
			return emitted, err
		}
		emitted++
	}
	return emitted, nil
}

func collectContacts(sellers []SellerContact) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range sellers {
		email := strings.ToLower(strings.TrimSpace(s.ContactEmail))
		if email == "" {
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
