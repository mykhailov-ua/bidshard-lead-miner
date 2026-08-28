package sourceregistry

import (
	"strings"

	"github.com/bidshard/parser/internal/metrics"
)

// CascadeTypes fans discovered affiliate domains into downstream crawlers.
var CascadeTypes = []string{TypeSupply, TypeLander, TypeTGWeb}

// AcceptCascadeDomain reports whether domain should enter the crawl cascade.
func AcceptCascadeDomain(domain string) bool {
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}
	if action, _, decided := HeuristicTriage(DomainMeta{Domain: domain}); decided && action == "drop" {
		return false
	}
	return true
}

// cascadePartnerDropHosts are SSP/CDN/ad-exchange rows common in ads.txt partner lists.
// Supply seeds may still crawl these directly; only partner fan-out is blocked.
var cascadePartnerDropHosts = map[string]struct{}{
	"rubiconproject.com":    {},
	"pubmatic.com":          {},
	"openx.com":             {},
	"appnexus.com":          {},
	"adnxs.com":             {},
	"indexexchange.com":     {},
	"casalemedia.com":       {},
	"triplelift.com":        {},
	"media.net":             {},
	"smartadserver.com":     {},
	"amazon-adsystem.com":   {},
	"doubleclick.net":       {},
	"googlesyndication.com": {},
	"googleadservices.com":  {},
	"adsrvr.org":            {},
	"criteo.com":            {},
	"taboola.com":           {},
	"outbrain.com":          {},
	"sharethrough.com":      {},
	"spotx.tv":              {},
	"yieldmo.com":           {},
	"contextweb.com":        {},
	"liveramp.com":          {},
	"adform.net":            {},
	"sovrn.com":             {},
	"lijit.com":             {},
	"33across.com":          {},
	"freewheel.tv":          {},
	"unrulymedia.com":       {},
	"conversantmedia.com":   {},
	"smaato.com":            {},
	"inmobi.com":            {},
	"chartboost.com":        {},
	"teads.tv":              {},
	"magnite.com":           {},
}

// AcceptCascadePartnerDomain filters ads.txt / sellers.json partner hosts before tgweb fan-out.
func AcceptCascadePartnerDomain(domain string) bool {
	if !AcceptCascadeDomain(domain) {
		return false
	}
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}
	if _, ok := cascadePartnerDropHosts[domain]; ok {
		return false
	}
	for blocked := range cascadePartnerDropHosts {
		if domain == blocked || strings.HasSuffix(domain, "."+blocked) {
			return false
		}
	}
	return true
}

// UpsertCascade registers domain for supply, lander, and tgweb crawls.
func UpsertCascade(path string, domain, discoveredBy, source, channel string) (added bool, err error) {
	domain = normalizeDomain(domain)
	if domain == "" || !AcceptCascadeDomain(domain) {
		return false, nil
	}
	types := append([]string(nil), CascadeTypes...)
	return Upsert(path, Entry{
		Domain:       domain,
		Types:        types,
		DiscoveredBy: strings.TrimSpace(discoveredBy),
		Source:       strings.TrimSpace(source),
		Channel:      strings.TrimSpace(channel),
	})
}

// CascadeDomains upserts many domains; returns count of newly added registry rows.
func CascadeDomains(path string, domains []string, discoveredBy, source string) (added int, err error) {
	for _, domain := range domains {
		ok, upsertErr := UpsertCascade(path, domain, discoveredBy, source, "")
		if upsertErr != nil {
			return added, upsertErr
		}
		if ok {
			added++
		}
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("domain_cascade", added)
	}
	return added, nil
}
