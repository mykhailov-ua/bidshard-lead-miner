package config

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PollInterval          time.Duration
	WorkerCount           int
	TaskBuffer            int
	SourceConcurrency     int
	ScanTimeout           time.Duration
	HTTPTimeout           time.Duration
	ShutdownTimeout       time.Duration
	LogFormat             string
	LogLevel              string
	Output                string
	WriteSlots            int
	ScanOnce              bool
	IngestStdin           bool
	IngestReader          io.Reader
	TelegramSidecar       bool
	TelegramDryRun        bool
	TelegramConfigPath    string
	Source                string
	SupplySeedPath        string
	SupplyHostRPS         float64
	SupplyMaxDomains      int
	SupplyBaseURL         string
	ForumSeedPath         string
	ForumBaseURL          string
	LanderSeedPath        string
	LanderBaseURL         string
	LanderHeadless        bool
	TelegramAPIID         int
	TelegramAPIHash       string
	MongoURI              string
	MongoDB               string
	MongoCollection       string
	ExportJSONPath        string
	MXCheck               bool
	KeywordsJSONPath      string
	KeywordsGrayPath      string
	GeoBlockCountries     []string
	DisposableDomainsPath string

	GeminiAPIKey          string
	GeminiModel           string
	GeminiAnalyzeInterval time.Duration
	GeminiReportInterval  time.Duration
	GeminiBatchSize       int
	GeminiRPM             int
	GeminiRPD             int
	GeminiTPM             int
	GeminiEmbedRPM        int
	GeminiMaxRetries      int
	GeminiRequestTimeout  time.Duration
	ColdJunkQueueSize     int
	ColdJunkCollection    string
	ColdReportCollection  string

	RedditSubreddits []string
	RedditQueries    []string
	RedditMaxResults int

	GeminiKeywordDiffEvery int
	GeminiKeywordDiffDir   string
	GeminiEmbedThreshold   float64
	ParserICPClassify      bool
	ParserGeoClassify      bool
	ParserTimeDecay        bool
	EnrichRDAP             bool
	EnrichDNS              bool
	EnrichEmail            bool
	EnrichSMTPVerify       bool
	DiscordBotToken        string
	DiscordChannelIDs      []string
	DiscordMaxMessages     int
	SourceStatsCollection  string
	CrmBoostCollection     string
	EmbeddingCollection    string

	BlacklistDomainsPath   string
	BlacklistEmailsPath    string
	ParserPilotTag         bool
	ParserLeadStatusEnabled bool
	KeywordsLocalePath     string
	KeywordsLocale         string
	CTQueries              []string
	CTMaxResults           int
	GitHubToken            string
	GitHubSearchQueries    []string
	MetricsAddr            string
	WarriorSeedPath        string
	WarriorHostRPS         float64
	KeywordStatsCollection string
	HTTPWorkers            int
	ProxyURLs              []string
}

func Load() (Config, error) {
	cfg := Config{
		PollInterval:           time.Duration(envInt("PARSER_POLL_SEC", 120)) * time.Second,
		WorkerCount:            envInt("PARSER_WORKERS", 4),
		TaskBuffer:             envInt("PARSER_TASK_BUFFER", 128),
		SourceConcurrency:      envInt("PARSER_SOURCE_CONCURRENCY", 3),
		ScanTimeout:            envDuration("PARSER_SCAN_TIMEOUT", 5*time.Minute),
		HTTPTimeout:            envDuration("PARSER_HTTP_TIMEOUT", 30*time.Second),
		ShutdownTimeout:        envDuration("PARSER_SHUTDOWN_TIMEOUT", 30*time.Second),
		HTTPWorkers:            envInt("PARSER_HTTP_WORKERS", 10),
		ProxyURLs:              parseCSV(env("PARSER_PROXY_LIST", "")),
		LogFormat:              env("PARSER_LOG_FORMAT", "json"),
		LogLevel:               env("PARSER_LOG_LEVEL", "info"),
		Output:                 env("PARSER_OUTPUT", "table"),
		WriteSlots:             envInt("PARSER_WRITE_SLOTS", 8),
		TelegramAPIHash:        env("TELEGRAM_API_HASH", ""),
		MongoURI:               env("MONGO_URI", ""),
		MongoDB:                env("MONGO_DB", "parser"),
		MongoCollection:        env("PARSER_MONGO_COLLECTION", "leads"),
		ExportJSONPath:         env("PARSER_EXPORT_JSON", ""),
		MXCheck:                envBool("PARSER_MX_CHECK", false),
		KeywordsJSONPath:       env("KEYWORDS_JSON_PATH", "data/keywords.json"),
		KeywordsGrayPath:       env("KEYWORDS_GRAY_JSON_PATH", "data/keywords-gray.json"),
		GeoBlockCountries:      parseCSV(env("GEO_BLOCK_COUNTRIES", "RU,BY")),
		DisposableDomainsPath:  env("DISPOSABLE_DOMAINS_PATH", "data/disposable_domains.txt"),
		TelegramConfigPath:     env("TELEGRAM_CONFIG_PATH", "config/sources.telegram.yaml"),
		Source:                 env("PARSER_SOURCE", "stub"),
		SupplySeedPath:         env("SUPPLY_SEED_PATH", "data/seeds/domains.csv"),
		SupplyHostRPS:          envFloat("SUPPLY_HOST_RPS", 2),
		SupplyBaseURL:          env("SUPPLY_BASE_URL", ""),
		ForumSeedPath:          env("FORUM_SEED_PATH", "data/seeds/forum_threads.csv"),
		ForumBaseURL:           env("FORUM_BASE_URL", ""),
		LanderSeedPath:         env("LANDER_SEED_PATH", "data/seeds/lander_urls.csv"),
		LanderBaseURL:          env("LANDER_BASE_URL", ""),
		LanderHeadless:         envBool("PARSER_LANDER_HEADLESS", false),
		GeminiAPIKey:           env("GEMINI_API_KEY", ""),
		GeminiModel:            env("GEMINI_MODEL", "gemini-2.0-flash"),
		GeminiAnalyzeInterval:  envDuration("GEMINI_ANALYZE_INTERVAL", 15*time.Minute),
		GeminiReportInterval:   envDuration("GEMINI_REPORT_INTERVAL", 6*time.Hour),
		GeminiBatchSize:        envInt("GEMINI_BATCH_SIZE", 20),
		GeminiRPM:              envInt("GEMINI_RPM", 0),
		GeminiRPD:              envInt("GEMINI_RPD", 0),
		GeminiTPM:              envInt("GEMINI_TPM", 0),
		GeminiEmbedRPM:         envInt("GEMINI_EMBED_RPM", 0),
		GeminiMaxRetries:       envInt("GEMINI_MAX_RETRIES", 0),
		GeminiRequestTimeout:   envDuration("GEMINI_REQUEST_TIMEOUT", 60*time.Second),
		ColdJunkQueueSize:      envInt("COLD_JUNK_QUEUE_SIZE", 512),
		ColdJunkCollection:     env("COLD_JUNK_COLLECTION", "junk_leads"),
		ColdReportCollection:   env("COLD_REPORT_COLLECTION", "junk_reports"),
		RedditSubreddits:       parseCSV(env("REDDIT_SUBREDDITS", "affiliatemarketing,media_buying,adops,PPC")),
		RedditQueries:          parseSemicolonCSV(env("REDDIT_QUERIES", "voluum alternative;tracker too expensive;postback failing;self-hosted tracker;tracker migration;switch from voluum")),
		RedditMaxResults:       envInt("REDDIT_MAX_RESULTS", 25),
		GeminiKeywordDiffEvery: envInt("GEMINI_KEYWORD_DIFF_EVERY", 5),
		GeminiKeywordDiffDir:   env("GEMINI_KEYWORD_DIFF_DIR", "data/suggestions"),
		GeminiEmbedThreshold:   envFloat("GEMINI_EMBED_THRESHOLD", 0.92),
		ParserICPClassify:      envBool("PARSER_ICP_CLASSIFY", true),
		ParserGeoClassify:      envBool("PARSER_GEO_CLASSIFY", true),
		ParserTimeDecay:        envBool("PARSER_TIME_DECAY", true),
		EnrichRDAP:             envBool("PARSER_ENRICH_RDAP", true),
		EnrichDNS:              envBool("PARSER_ENRICH_DNS", true),
		EnrichEmail:            envBool("PARSER_ENRICH_EMAIL", true),
		EnrichSMTPVerify:       envBool("PARSER_ENRICH_SMTP_VERIFY", false),
		DiscordBotToken:        env("DISCORD_BOT_TOKEN", ""),
		DiscordChannelIDs:      parseCSV(env("DISCORD_CHANNEL_IDS", "")),
		DiscordMaxMessages:     envInt("DISCORD_MAX_MESSAGES", 50),
		SourceStatsCollection:  env("SOURCE_STATS_COLLECTION", "source_stats"),
		CrmBoostCollection:     env("CRM_BOOST_COLLECTION", "crm_boosts"),
		EmbeddingCollection:    env("EMBEDDING_COLLECTION", "snippet_embeddings"),
		BlacklistDomainsPath:   env("BLACKLIST_DOMAINS_PATH", "data/blacklist_domains.txt"),
		BlacklistEmailsPath:    env("BLACKLIST_EMAILS_PATH", "data/blacklist_emails.txt"),
		ParserPilotTag:         envBool("PARSER_PILOT_TAG", true),
		ParserLeadStatusEnabled: envBool("PARSER_LEAD_STATUS_ENABLED", false),
		KeywordsLocalePath:     env("KEYWORDS_LOCALE_PATH", ""),
		KeywordsLocale:         env("KEYWORDS_LOCALE", ""),
		CTQueries:              parseCSV(env("CT_QUERIES", "track,click,go")),
		CTMaxResults:           envInt("CT_MAX_RESULTS", 100),
		GitHubToken:            env("GITHUB_TOKEN", ""),
		GitHubSearchQueries:    parseSemicolonCSV(env("GITHUB_SEARCH_QUERIES", "voluum alternative;self-hosted tracker;keitaro docker")),
		MetricsAddr:            env("PARSER_METRICS_ADDR", ""),
		WarriorSeedPath:        env("WARRIOR_SEED_PATH", "data/seeds/warrior_threads.csv"),
		WarriorHostRPS:         envFloat("WARRIOR_HOST_RPS", 1),
		KeywordStatsCollection: env("KEYWORD_STATS_COLLECTION", "keyword_stats"),
	}

	apiID, err := envIntOptional("TELEGRAM_API_ID")
	if err != nil {
		return Config{}, fmt.Errorf("TELEGRAM_API_ID: %w", err)
	}
	cfg.TelegramAPIID = apiID

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envIntOptional(key string) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseSemicolonCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
