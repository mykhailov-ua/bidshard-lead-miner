package filter

import (
	"regexp"
	"strings"
)

var sellerAuthorTokenRe = regexp.MustCompile(`(?i)(?:^|[\s._-])(shop|store|rent|agency|seller|support|storefront)(?:[\s._-]|$)`)

var sellerAuthorCyrillic = []string{
	"магазин", "аренда", "агентство", "селлер", "саппорт", "поддержка",
	"продажа аккаунтов", "accounts shop", "account store",
}

// SellerAuthorProfile is true when username or channel bio looks like a seller storefront.
func SellerAuthorProfile(username, channelAbout, title string) (bool, string) {
	combined := strings.TrimSpace(username + " " + channelAbout + " " + title)
	if combined == "" {
		return false, ""
	}
	lower := strings.ToLower(combined)
	if sellerAuthorTokenRe.MatchString(lower) || hasSellerAuthorToken(lower) {
		return true, "seller author profile token"
	}
	for _, phrase := range sellerAuthorCyrillic {
		if strings.Contains(lower, phrase) {
			return true, "seller author profile: " + phrase
		}
	}
	return false, ""
}

func hasSellerAuthorToken(lower string) bool {
	for _, part := range strings.FieldsFunc(lower, func(r rune) bool {
		return r == '_' || r == '.' || r == '-' || r == ' ' || r == '@'
	}) {
		switch strings.TrimSpace(part) {
		case "shop", "store", "rent", "agency", "seller", "support", "storefront":
			return true
		}
	}
	return false
}

var technicalAuthorSignals = []string{
	"nginx", "htop", "upstream timed out", "worker_connections",
	"error.log", "access.log", "stack trace", "stacktrace",
	"502 bad gateway", "504 gateway timeout", "connection reset by peer",
	"ssl handshake", "too many open files", "oom killed", "out of memory",
	"clickhouse", "mysql lock", "redis timeout", "php-fpm",
	"server {", "location /", "proxy_pass", "fastcgi_pass",
	"grep error", "tail -f", "journalctl", "dmesg",
	"cpu 100%", "load average", "iowait", "swap usage",
}

var teamBuyingSignals = []string{
	"media buying team", "team of buyers", "multiple buyers", "buying team",
	"multi-user keitaro", "keitaro team", "binom team", "our media buyers",
	"head of acquisition", "team lead media", "affiliate director",
	"self-hosted tracker", "self hosted tracker", "on our vps", "our vps",
	"own server", "our server", "dedicated vps",
	"$30k", "$50k", "30k budget", "50k spend", "30k/month", "50k/month",
	"parallel pilot", "running keitaro", "running binom",
}

// TeamBuyingSignal reports in-house team scale (not solo seller or course funnel).
func TeamBuyingSignal(text, title string) bool {
	combined := strings.TrimSpace(title + " " + text)
	if combined == "" {
		return false
	}
	lower := strings.ToLower(combined)
	hits := 0
	for _, sig := range teamBuyingSignals {
		if strings.Contains(lower, sig) {
			hits++
			if hits >= 2 {
				return true
			}
		}
	}
	// One strong team signal plus tracker pain.
	if hits >= 1 {
		for _, pain := range []string{"keitaro", "binom", "voluum", "postback", "tracker", "self-hosted"} {
			if strings.Contains(lower, pain) {
				return true
			}
		}
	}
	return false
}

// TechnicalAuthorSignal reports long-form infra debugging (DevOps / technical founder).
func TechnicalAuthorSignal(text, title string) bool {
	combined := strings.TrimSpace(title + " " + text)
	if runeLen(combined) < 120 {
		return false
	}
	lower := strings.ToLower(combined)
	hits := 0
	for _, sig := range technicalAuthorSignals {
		if strings.Contains(lower, sig) {
			hits++
			if hits >= 2 {
				return true
			}
		}
	}
	// Single strong signal + question or log paste.
	if hits >= 1 && (strings.Contains(lower, "?") || strings.Contains(lower, "error") || strings.Contains(lower, "config")) {
		return runeLen(combined) >= 200
	}
	return false
}
