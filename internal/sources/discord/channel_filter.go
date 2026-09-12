package discord

import "strings"

var discordSkipChannelNames = []string{
	"rules", "welcome", "admin", "staff", "mod", "nsfw", "verify",
	"verification", "announcements-only", "bot", "commands",
}

var discordICPTokens = []string{
	"affiliate", "igaming", "tracker", "voluum", "keitaro", "binom", "redtrack",
	"media", "buying", "arbitrage", "postback", "cpa", "gambling", "betting",
	"igaming", "pwa", "funnel", "traffic", "offer", "network",
}

// InviteEntryLooksICP trusts SERP/catalog hints stored in invite registry.
func InviteEntryLooksICP(inv InviteEntry) bool {
	hint := strings.TrimSpace(inv.GuildHint + " " + inv.Query)
	if hint == "" {
		return false
	}
	return GuildLooksICP(hint, inv.Source)
}

// GuildLooksICP filters server names/hints from public catalogs.
func GuildLooksICP(name, hint string) bool {
	text := strings.ToLower(strings.TrimSpace(name + " " + hint))
	if text == "" {
		return false
	}
	if status, _ := HeuristicTriageInvite("", text); status == "drop" {
		return false
	}
	for _, tok := range discordICPTokens {
		if strings.Contains(text, tok) {
			return true
		}
	}
	return false
}

// ChannelLooksReadable reports text channels worth scraping.
func ChannelLooksReadable(name string, channelType int) bool {
	if channelType != 0 && channelType != 5 {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return false
	}
	for _, skip := range discordSkipChannelNames {
		if lower == skip || strings.HasPrefix(lower, skip+"-") {
			return false
		}
	}
	for _, tok := range discordICPTokens {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	// General chat buckets on ICP guilds.
	for _, tok := range []string{"general", "chat", "discussion", "support", "help", "off-topic", "offtopic", "main"} {
		if strings.Contains(lower, tok) {
			return true
		}
	}
	return false
}
