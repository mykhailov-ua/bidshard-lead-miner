package sources

// SourceInfo describes a crawl source for CLI listing and config checks.
type SourceInfo struct {
	Name     string
	InAll    bool
	Requires []string
	Note     string
}

// Catalog returns registered source names and their env prerequisites.
func Catalog() []SourceInfo {
	return []SourceInfo{
		{Name: "forum", InAll: true, Requires: []string{"FORUM_SEED_PATH"}},
		{Name: "supply", InAll: true, Requires: []string{"SUPPLY_SEED_PATH"}},
		{Name: "lander", Requires: []string{"LANDER_SEED_PATH"}, Note: "opt-in; not in all (competitor lander HTML noise)"},
		{Name: "reddit", InAll: true, Note: "PullPush API; may rate-limit"},
		{Name: "discord", InAll: true, Requires: []string{"DISCORD_BOT_TOKEN", "DISCORD_CHANNEL_IDS"}},
		{Name: "warrior", Note: "deprecated alias for forum; warriorforum.com via WARRIOR_SEED_PATH"},
		{Name: "ct", Note: "opt-in only; not in all"},
		{Name: "github", Requires: []string{"GITHUB_TOKEN"}, Note: "opt-in only"},
		{Name: "serp", InAll: true, Note: "DuckDuckGo dorks; writes data/runtime/discovered_telegram_channels.json"},
		{Name: "webpain", Note: "opt-in; crawl data/runtime/discovered_web_pain.json from open-web SERP"},
	}
}
