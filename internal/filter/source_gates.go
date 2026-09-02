package filter

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

var buyerIntentPhrases = []string{
	"looking for", "need a tracker", "need tracker", "recommend", "alternative to",
	"migrate", "switching from", "anyone facing", "help with", "does anyone",
	"subscription plan", "hiring", "media buyer", "clients looking",
}

var githubVendorOrgs = map[string]struct{}{
	"keitaroinc": {}, "voluum": {}, "binom": {}, "redtrack": {}, "clickflare": {},
}

var githubTrackerPainHints = []string{
	"postback", "s2s postback", "s2s", "voluum alternative", "keitaro alternative",
	"tracker alternative", "media buyer", "affiliate marketing", "igaming affiliate",
	"cost sync", "clickid", "click id", "conversion tracking", "switching from",
	"migrate from", "moving off voluum", "budget protection", "ftd", "cloak",
	"not working", "failing", "expensive", "billing", "spend",
}

// IsLanderSource reports competitor lander HTML crawl items.
func IsLanderSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "lander:")
}

// IsWebPainSource reports SERP-harvested web pain crawl items.
func IsWebPainSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "webpain:")
}

// IsGitHubSource reports GitHub issue search items.
func IsGitHubSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "github:")
}

// LanderDomainFromSource returns hostname from lander:example.com items.
func LanderDomainFromSource(source string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(source)), "lander:")
}

// LanderBlacklistedSource rejects competitor tracker marketing pages on crawl blacklist.
func LanderBlacklistedSource(source string) bool {
	if !IsLanderSource(source) {
		return false
	}
	return validate.IsBlacklistedDomain(LanderDomainFromSource(source))
}

// LanderRequiresBuyerSignal rejects marketing-page keyword noise without buyer language.
func LanderRequiresBuyerSignal(text string) bool {
	return validate.HasCommercialPainIntent(text) || validate.HasBuyerQuestionPattern(text) || scoring.HasBuyerIntentSignal(text)
}

// LanderRequiresEmailOrSkype rejects CSS @media-only lander scrapes.
func LanderRequiresEmailOrSkype(contacts []extract.Contact) bool {
	for _, c := range contacts {
		if c.Type == "email" || c.Type == "skype" {
			return true
		}
	}
	return false
}

// GitHubRequiresPainContext blocks infra/vendor issues without buyer or tracker pain.
func GitHubRequiresPainContext(source, text string) bool {
	if !IsGitHubSource(source) {
		return true
	}
	if GitHubVendorOrg(source) {
		return false
	}
	if scoring.HasBuyerIntentSignal(text) {
		return true
	}
	tier, _ := scoring.DetectDisplacementTier(text, nil)
	if tier != scoring.DisplacementNone {
		return true
	}
	lower := strings.ToLower(text)
	if scoring.HasCompetitorMention(lower) && hasGitHubTrackerPain(lower) {
		return true
	}
	return false
}

func GitHubVendorOrg(source string) bool {
	rest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(source)), "github:")
	if idx := strings.Index(rest, "/"); idx > 0 {
		_, blocked := githubVendorOrgs[rest[:idx]]
		return blocked
	}
	return false
}

func hasGitHubTrackerPain(lower string) bool {
	for _, hint := range githubTrackerPainHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// TelegramChannelSelfBroadcast is true when the only telegram contact is the channel itself.
func TelegramChannelSelfBroadcast(source string, contacts []extract.Contact) bool {
	if !isTelegramSource(source) {
		return false
	}
	channel := telegramChannelFromSource(source)
	if channel == "" {
		return false
	}
	for _, c := range contacts {
		switch c.Type {
		case "email", "skype", "forum_user", "reddit", "github":
			return false
		case "telegram":
			handle := normalizeTelegramHandle(c.Value)
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Value)), "user_id:") {
				return false
			}
			if handle == "" || extract.IsJunkTelegramHandle(c.Value) {
				continue
			}
			if handle != channel {
				return false
			}
		}
	}
	return true
}

var telegramIntelOnlyHandles = map[string]struct{}{
	"igaming_news":             {},
	"partnerkin_job":           {},
	"affiliatechannel_igaming": {},
	"partneroff_pro":           {},
}

var telegramIntelOnlySubstrings = []string{
	"_news", "_jobs", "_job_", "jobboard", "job_board", "vacancy",
}

// TelegramIntelOnlyChannel reports news digests and job-board channels (not buyer dialog).
func TelegramIntelOnlyChannel(source string) bool {
	if !isTelegramSource(source) {
		return false
	}
	channel := telegramChannelFromSource(source)
	if channel == "" {
		return false
	}
	if _, ok := telegramIntelOnlyHandles[channel]; ok {
		return true
	}
	for _, sub := range telegramIntelOnlySubstrings {
		if strings.Contains(channel, sub) {
			return true
		}
	}
	return false
}

var vacancyPhrases = []string{
	"шукаємо", "ваканс", "ищем", "hiring", "job opening", "join our team",
	"remote |", "full-time", "full time", "recruiter", "head of media buying",
	"media buyer fb", "tiktok media buyer", "salary", "зарплат", "k+/month",
	"$k+/", "profit share", "бонус", "офіс", "office)",
}

// TelegramInviteWithoutBuyerIntent blocks invite-channel promos without buyer pain/intent.
func TelegramInviteWithoutBuyerIntent(source, text string) bool {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "telegram:invite:") {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	if telegramInviteVacancyNoise(lower) {
		return true
	}
	if hasPainKeyword(lower) || hasBuyerIntentPhrase(lower) {
		return false
	}
	return true
}

func telegramInviteVacancyNoise(lower string) bool {
	for _, phrase := range vacancyPhrases {
		if !strings.Contains(lower, phrase) {
			continue
		}
		if hasPainKeyword(lower) || hasTrackerPainHint(lower) {
			return false
		}
		return true
	}
	return false
}

func hasTrackerPainHint(lower string) bool {
	for _, hint := range githubTrackerPainHints {
		switch hint {
		case "media buyer", "affiliate marketing", "igaming affiliate":
			continue
		}
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

func telegramChannelFromSource(source string) string {
	source = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(source)), "telegram:")
	if strings.HasPrefix(source, "invite:") {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(source), "@")
}

func normalizeTelegramHandle(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "@")))
}

func hasBuyerIntentPhrase(lower string) bool {
	for _, phrase := range buyerIntentPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	if strings.Contains(lower, "?") && runeLen(lower) >= minTelegramTextRunes {
		return true
	}
	return false
}
