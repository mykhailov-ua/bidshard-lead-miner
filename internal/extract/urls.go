package extract

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

var blockedWebHosts = map[string]struct{}{
	"t.me":                      {},
	"telegram.me":               {},
	"telegram.org":              {},
	"instagram.com":             {},
	"facebook.com":              {},
	"fb.com":                    {},
	"twitter.com":               {},
	"x.com":                     {},
	"youtube.com":               {},
	"youtu.be":                  {},
	"google.com":                {},
	"linkedin.com":              {},
	"bit.ly":                    {},
	"tiktok.com":                {},
	"discord.gg":                {},
	"discord.com":               {},
	"wa.me":                     {},
	"whatsapp.com":              {},
	"forms.gle":                 {},
	"goo.gl":                    {},
	"linktr.ee":                 {},
	"taplink.cc":                {},
	"vk.com":                    {},
	"ok.ru":                     {},
	"medium.com":                {},
	"github.com":                {},
	"notion.site":               {},
	"docs.google.com":           {},
	"magiceden.io":              {},
	"jup.ag":                    {},
	"raydium.io":                {},
	"knowyourmeme.com":          {},
	"pump.fun":                  {},
	"collab.land":               {},
	"telegra.ph":                {},
	"teletype.in":               {},
	"chatgpt.com":               {},
	"openai.com":                {},
	"binance.com":               {},
	"bing.com":                  {},
	"wikipedia.org":             {},
	"reddit.com":                {},
	"amazon.com":                {},
	"reuters.com":               {},
	"gov.uk":                    {},
	"legislation.gov.uk":        {},
	"gamblingcommission.gov.uk": {},
	"niesr.ac.uk":               {},
	"icann.org":                 {},
	"cloudflare.com":            {},
	"sentry.io":                 {},
	"google.co.uk":              {},
	"googleapis.com":            {},
	"gstatic.com":               {},
	"site.com":                  {},
	"site2.com":                 {},
	"us.com":                    {},
	"us.net":                    {},
	"it.com":                    {},
}

var (
	urlRe      = regexp.MustCompile(`https?://[^\s<>"']+`)
	bareHostRe = regexp.MustCompile(`(?i)(?:^|[\s"'<(])((?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,})(?:[^\w.-]|$)`)
)

// WebDomains extracts registrable hostnames from URLs and bare domains in text.
func WebDomains(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(host string) {
		host = normalizeWebHost(host)
		if host == "" || isBlockedWebHost(host) || !IsValidWebDomain(host) {
			return
		}
		if _, ok := seen[host]; ok {
			return
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}

	for _, m := range urlRe.FindAllString(text, -1) {
		if u, err := url.Parse(m); err == nil && u.Host != "" {
			add(u.Host)
		}
	}
	for _, m := range bareHostRe.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return dropSuffixFragments(dropHyphenFragments(text, out))
}

func dropHyphenFragments(text string, hosts []string) []string {
	lower := strings.ToLower(text)
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		if strings.Contains(lower, "-"+host) {
			continue
		}
		out = append(out, host)
	}
	return out
}

func dropSuffixFragments(hosts []string) []string {
	if len(hosts) < 2 {
		return hosts
	}
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		skip := false
		for _, other := range hosts {
			if host != other && strings.HasSuffix(other, "."+host) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, host)
		}
	}
	return out
}

func normalizeWebHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.Trim(host, ".")
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	return host
}

func isBlockedWebHost(host string) bool {
	return IsBlockedWebHost(host)
}

// IsBlockedWebHost reports social, search, gov, and configured blacklist hosts.
func IsBlockedWebHost(host string) bool {
	host = normalizeWebHost(host)
	if host == "" {
		return true
	}
	if _, ok := blockedWebHosts[host]; ok {
		return true
	}
	if validate.IsBlacklistedDomain(host) {
		return true
	}
	for blocked := range blockedWebHosts {
		if strings.HasSuffix(host, "."+blocked) {
			return true
		}
	}
	return strings.HasSuffix(host, ".bot") || strings.HasSuffix(host, ".local")
}

// IsValidWebDomain reports whether host looks like a real site (not file path, invalid TLD).
func IsValidWebDomain(host string) bool {
	host = normalizeWebHost(host)
	if host == "" || IsBlockedWebHost(host) {
		return false
	}
	invalidSuffixes := []string{
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg",
		".pdf", ".txt", ".html", ".htm", ".php", ".css", ".js", ".json", ".xml",
	}
	for _, suf := range invalidSuffixes {
		if strings.HasSuffix(host, suf) {
			return false
		}
	}
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return false
	}
	tld := parts[len(parts)-1]
	if len(tld) < 2 || len(tld) > 24 {
		return false
	}
	invalidTLD := map[string]struct{}{
		"gram": {}, "ggate": {}, "dxb": {}, "txt": {}, "html": {}, "htm": {},
		"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "php": {}, "pdf": {},
		"local": {}, "bot": {},
	}
	if _, ok := invalidTLD[tld]; ok {
		return false
	}
	for _, label := range parts {
		if label == "" {
			return false
		}
	}
	return true
}
