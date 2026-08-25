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
	PollInterval              time.Duration
	WorkerCount               int
	TaskBuffer                int
	SourceConcurrency         int
	ScanTimeout               time.Duration
	HTTPTimeout               time.Duration
	ShutdownTimeout           time.Duration
	CollectDrainTimeout       time.Duration
	LogFormat                 string
	LogLevel                  string
	Output                    string
	WriteSlots                int
	ScanOnce                  bool
	IngestStdin               bool
	IngestReader              io.Reader
	TelegramSidecar           bool
	TelegramDryRun            bool
	TelegramConfigPath        string
	TelethonPython            string
	Source                    string
	SupplySeedPath            string
	SupplyHostRPS             float64
	SupplyMaxDomains          int
	SupplyBaseURL             string
	SourceRegistryPath        string
	ForumSeedPath             string
	ForumRegistryPath         string
	ForumBaseURL              string
	LanderSeedPath            string
	LanderBaseURL             string
	LanderHeadless            bool
	LanderHeadlessDefer       bool
	LanderHeadlessQueuePath   string
	LanderHeadlessQueueMax    int
	LanderHeadlessDrainLimit  int
	LanderHeadlessMaxBrowsers int
	TelegramAPIID             int
	TelegramAPIHash           string
	MongoURI                  string
	MongoDB                   string
	MongoCollection           string
	ExportJSONPath            string
	ExportJSONFormat          string
	MXCheck                   bool
	KeywordsJSONPath          string
	KeywordsGrayPath          string
	GeoBlockCountries         []string
	DisposableDomainsPath     string

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

	GeminiKeywordDiffEvery          int
	GeminiKeywordDiffDir            string
	GeminiDiscoverDiffEvery         int
	GeminiDiscoverDiffDir           string
	GeminiPainVocabDiffEvery        int
	GeminiEmbedThreshold            float64
	HardRejectShadowPct             int
	HardRejectShadowDailyCap        int
	StaleLeadRegradeInterval        time.Duration
	StaleLeadAge                    time.Duration
	DuplicateSuggestInterval        time.Duration
	DuplicateSuggestWindow          time.Duration
	GeoAuditInterval                time.Duration
	GeoAuditSampleN                 int
	GeoAuditCollection              string
	WebhookAuditInterval            time.Duration
	WebhookFeedbackCollection       string
	ParserChannelTriage             bool
	BGChannelTriageInterval         time.Duration
	ParserDomainTriage              bool
	ParserLanderPathDiscover        bool
	ParserLanderPathGemini          bool
	LanderPathsCachePath            string
	BGDomainTriageInterval          time.Duration
	DomainTriageCachePath           string
	TelegramChannelsPath            string
	GeminiEmbedPainMin              float64
	GeminiEmbedSpamMin              float64
	GeminiLeadAnalyzeInterval       time.Duration
	GeminiLeadBatchSize             int
	GeminiQuotaCriticalPct          int
	GeminiQuotaHighPct              int
	GeminiQuotaNormalPct            int
	GeminiQuotaLowPct               int
	WarmLeadQueueSize               int
	WarmAnalysisRetryMax            int           // Gemini batch retries before DLQ (WARM_ANALYSIS_RETRY_MAX)
	WarmAnalysisRetryBase           time.Duration // Initial retry backoff (WARM_ANALYSIS_RETRY_BASE)
	WarmAnalysisPendingScanInterval time.Duration // Mongo pending rescan period (WARM_ANALYSIS_PENDING_SCAN_INTERVAL)
	WarmAnalysisPendingStale        time.Duration // Min pending age before rescan (WARM_ANALYSIS_PENDING_STALE)
	WarmAnalysisDLQCollection       string        // Empty disables DLQ writes
	WarmAnalysisShutdownDrain       time.Duration // Warm-path flush budget on shutdown (WARM_ANALYSIS_SHUTDOWN_DRAIN)
	ParserICPClassify               bool
	ParserICPClassifyTgWeb          bool   // sync ICP on tgweb hot path even when ParserGeminiDefer is true
	ParserTgWebPrescanMode          string // aggressive: site LPR bypasses keyword prescan; strict: require affiliate hits
	ParserGeoClassify               bool
	ParserTimeDecay                 bool
	EnrichRDAP                      bool
	EnrichDNS                       bool
	EnrichEmail                     bool
	EnrichSMTPVerify                bool
	DiscordBotToken                 string
	DiscordChannelIDs               []string
	DiscordMaxMessages              int
	DiscordRegistryPath             string
	SourceStatsCollection           string
	CrmBoostCollection              string
	EmbeddingCollection             string

	BlacklistDomainsPath              string
	BlacklistEmailsPath               string
	ParserPilotTag                    bool
	ParserGeminiEngage                bool
	ParserEntityOutreachNarrative     bool
	ParserEmbedPrescan                bool
	ParserWarmEmbedPrescan            bool
	ParserEmbedCluster                bool
	ParserWarmEmbedCluster            bool
	ParserGeminiEngageMedium          bool
	ParserGeminiEnrichSynth           bool
	ParserGeminiDefer                 bool // batch geo/ICP/engage/enrich on warm path; see deps.go geminiDefer matrix
	ParserGeminiSyncGeo               bool // inline geo on hot path when defer=true (PARSER_GEMINI_SYNC_GEO)
	ParserLeadStatusEnabled           bool
	KeywordsLocalePath                string
	KeywordsLocale                    string
	CTQueries                         []string
	CTMaxResults                      int
	GitHubToken                       string
	GitHubSearchQueries               []string
	GitHubRotateEnabled               bool
	GitHubRotateStatePath             string
	MetricsAddr                       string
	CRMWebhookURL                     string
	CRMWebhookSecret                  string
	CRMWebhookEnabled                 bool
	CRMWebhookAfterAnalysis           bool // defer mode: CRM webhook from warm path on analysis_status=done only
	CRMWebhookHeatMin                 string
	WarriorSeedPath                   string
	WarriorHostRPS                    float64
	KeywordStatsCollection            string
	ParserEntitySightings             bool
	EntityCollection                  string
	ParserCrossSourceHot              bool
	CrossSourceHotWindow              time.Duration
	CrossSourceHotBoost               int
	ParserEntityHeatEnabled           bool
	EntityHeatBlazing                 float64
	EntityHeatHot                     float64
	EntityHeatWarm                    float64
	EntityHeatDecay7D                 float64
	EntityHeatDecay30D                float64
	EntityHeatDecay90D                float64
	ParserEntityGeminiEnabled         bool
	EntityGeminiDebounce              time.Duration
	EntityGeminiLowConfidenceDebounce time.Duration
	EntityGeminiInterval              time.Duration
	EntityGeminiQueueSize             int
	ParserEntityLinkSuggest           bool
	EntityLinkSuggestInterval         time.Duration
	HTTPWorkers                       int
	ProxyURLs                         []string // PARSER_PROXY_LIST; HTTP crawlers only (not Mongo/Gemini)
	ProxySources                      []string // PARSER_PROXY_SOURCES; empty = all crawlers may use proxy
	ProxyDailyMBCap                   int      // PARSER_PROXY_DAILY_MB_CAP; 0 = no cap
	ProxyBudgetStatePath              string
	TelegramProxyURL                  string // TELEGRAM_PROXY_URL; MTProto sidecar only (socks5/http)
	BGWorkerEnabled                   bool
	BGTelegramEnabled                 bool
	BGSerpTelegramInterval            time.Duration
	BGTelegramDiscoverInterval        time.Duration
	BGTelegramScrapeInterval          time.Duration
	BGTelegramWebInterval             time.Duration
	BGForumDiscoverInterval           time.Duration
	BGSourceRegistrySyncInterval      time.Duration
	BGAutoReportInterval              time.Duration
	BGDiscordDiscoverInterval         time.Duration
	BGSourceDisableInterval           time.Duration
	AutoReportPath                    string
	DisabledSourcesPath               string
	SourceDisableMinRaw               int
	ParserSourceDisableGovernor       bool
	ParserAutoDiscover                bool
	ParserSeedFeedback                bool
	ParserSeedFeedbackMinHeat         string
	DiscoverAutoApplyMaxWeek          int
	TelegramDomainsPath               string
	TelegramWebMaxDomains             int
	TelegramWebRescanDays             int
	TelegramWebDomains                []string // allowlist; empty = pending queue from registry
	ProcessorTaskTimeout              time.Duration
}

func Load() (Config, error) {
	LoadDotEnv()
	// Empty PARSER_PROXY_LIST means direct egress. Comma-separated HTTP proxy URLs; no shell expansion.
	// GeminiRPM/RPD/TPM/EmbedRPM/MaxRetries at 0 -> gemini.NewClient applies model-tier defaults.
	cfg := Config{
		PollInterval:                      time.Duration(envInt("PARSER_POLL_SEC", 120)) * time.Second,
		WorkerCount:                       envInt("PARSER_WORKERS", 4),
		TaskBuffer:                        envInt("PARSER_TASK_BUFFER", 128),
		SourceConcurrency:                 envInt("PARSER_SOURCE_CONCURRENCY", 3),
		ScanTimeout:                       envDuration("PARSER_SCAN_TIMEOUT", 5*time.Minute),
		HTTPTimeout:                       envDuration("PARSER_HTTP_TIMEOUT", 30*time.Second),
		ShutdownTimeout:                   envDuration("PARSER_SHUTDOWN_TIMEOUT", 120*time.Second),
		CollectDrainTimeout:               envDuration("PARSER_COLLECT_DRAIN_TIMEOUT", 120*time.Second),
		HTTPWorkers:                       envInt("PARSER_HTTP_WORKERS", 10),
		ProxyURLs:                         parseCSV(env("PARSER_PROXY_LIST", "")),
		ProxySources:                      parseCSV(strings.ToLower(env("PARSER_PROXY_SOURCES", ""))),
		ProxyDailyMBCap:                   envInt("PARSER_PROXY_DAILY_MB_CAP", 0),
		ProxyBudgetStatePath:              env("PARSER_PROXY_BUDGET_STATE_PATH", "data/runtime/proxy_budget.json"),
		LogFormat:                         env("PARSER_LOG_FORMAT", "auto"),
		LogLevel:                          env("PARSER_LOG_LEVEL", "info"),
		Output:                            env("PARSER_OUTPUT", "auto"),
		WriteSlots:                        envInt("PARSER_WRITE_SLOTS", 8),
		TelegramAPIHash:                   env("TELEGRAM_API_HASH", ""),
		MongoURI:                          env("MONGO_URI", ""),
		MongoDB:                           env("MONGO_DB", "parser"),
		MongoCollection:                   env("PARSER_MONGO_COLLECTION", "leads"),
		ExportJSONPath:                    env("PARSER_EXPORT_JSON", ""),
		ExportJSONFormat:                  env("PARSER_EXPORT_JSON_FORMAT", "auto"),
		MXCheck:                           envBool("PARSER_MX_CHECK", false),
		KeywordsJSONPath:                  env("KEYWORDS_JSON_PATH", "data/keywords.json"),
		KeywordsGrayPath:                  env("KEYWORDS_GRAY_JSON_PATH", "data/keywords-gray.json"),
		GeoBlockCountries:                 parseCSV(env("GEO_BLOCK_COUNTRIES", "RU,BY")),
		DisposableDomainsPath:             env("DISPOSABLE_DOMAINS_PATH", "data/disposable_domains.txt"),
		TelegramConfigPath:                env("TELEGRAM_CONFIG_PATH", "config/sources.telegram.yaml"),
		TelethonPython:                    env("PARSER_TELETHON_PYTHON", ""),
		Source:                            env("PARSER_SOURCE", "all"),
		SupplySeedPath:                    env("SUPPLY_SEED_PATH", "data/seeds/domains.csv"),
		SupplyHostRPS:                     envFloat("SUPPLY_HOST_RPS", 2),
		SupplyBaseURL:                     env("SUPPLY_BASE_URL", ""),
		SourceRegistryPath:                env("SOURCE_REGISTRY_PATH", "data/runtime/source_registry.json"),
		ForumSeedPath:                     env("FORUM_SEED_PATH", "data/seeds/forum_threads.csv"),
		ForumRegistryPath:                 env("FORUM_REGISTRY_PATH", "data/runtime/discovered_forum_threads.json"),
		ForumBaseURL:                      env("FORUM_BASE_URL", ""),
		LanderSeedPath:                    env("LANDER_SEED_PATH", "data/seeds/lander_urls.csv"),
		LanderBaseURL:                     env("LANDER_BASE_URL", ""),
		LanderHeadless:                    envBool("PARSER_LANDER_HEADLESS", false),
		LanderHeadlessDefer:               envBool("PARSER_LANDER_HEADLESS_DEFER", false),
		LanderHeadlessQueuePath:           env("PARSER_LANDER_HEADLESS_QUEUE_PATH", "data/runtime/headless_queue.json"),
		LanderHeadlessQueueMax:            envInt("PARSER_LANDER_HEADLESS_QUEUE_MAX", 200),
		LanderHeadlessDrainLimit:          envInt("PARSER_LANDER_HEADLESS_DRAIN_LIMIT", 25),
		LanderHeadlessMaxBrowsers:         envInt("PARSER_LANDER_HEADLESS_MAX_BROWSERS", 2),
		GeminiAPIKey:                      env("GEMINI_API_KEY", ""),
		GeminiModel:                       env("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiAnalyzeInterval:             envDuration("GEMINI_ANALYZE_INTERVAL", 15*time.Minute),
		GeminiReportInterval:              envDuration("GEMINI_REPORT_INTERVAL", 6*time.Hour),
		GeminiBatchSize:                   envInt("GEMINI_BATCH_SIZE", 20),
		GeminiRPM:                         envInt("GEMINI_RPM", 0),
		GeminiRPD:                         envInt("GEMINI_RPD", 0),
		GeminiTPM:                         envInt("GEMINI_TPM", 0),
		GeminiEmbedRPM:                    envInt("GEMINI_EMBED_RPM", 0),
		GeminiMaxRetries:                  envInt("GEMINI_MAX_RETRIES", 0),
		GeminiRequestTimeout:              envDuration("GEMINI_REQUEST_TIMEOUT", 60*time.Second),
		ColdJunkQueueSize:                 envInt("COLD_JUNK_QUEUE_SIZE", 512),
		ColdJunkCollection:                env("COLD_JUNK_COLLECTION", "junk_leads"),
		ColdReportCollection:              env("COLD_REPORT_COLLECTION", "junk_reports"),
		RedditSubreddits:                  parseCSV(env("REDDIT_SUBREDDITS", "affiliatemarketing,media_buying,adops,PPC,juststart")),
		RedditQueries:                     parseSemicolonCSV(env("REDDIT_QUERIES", "voluum alternative;tracker too expensive;postback failing;click id not found;subid not found;tracker numbers don't match;conversion tracking discrepancy;self-hosted tracker;tracker migration;switch from voluum")),
		RedditMaxResults:                  envInt("REDDIT_MAX_RESULTS", 25),
		GeminiKeywordDiffEvery:            envInt("GEMINI_KEYWORD_DIFF_EVERY", 5),
		GeminiKeywordDiffDir:              env("GEMINI_KEYWORD_DIFF_DIR", "data/suggestions"),
		GeminiDiscoverDiffEvery:           envInt("GEMINI_DISCOVER_DIFF_EVERY", 0),
		GeminiDiscoverDiffDir:             env("GEMINI_DISCOVER_DIFF_DIR", ""),
		GeminiPainVocabDiffEvery:          envInt("GEMINI_PAIN_VOCAB_DIFF_EVERY", 0),
		GeminiEmbedThreshold:              envFloat("GEMINI_EMBED_THRESHOLD", 0.92),
		HardRejectShadowPct:               envInt("PARSER_HARD_REJECT_SHADOW_PCT", 2),
		HardRejectShadowDailyCap:          envInt("PARSER_HARD_REJECT_SHADOW_DAILY_CAP", 10),
		StaleLeadRegradeInterval:          envDuration("PARSER_STALE_LEAD_REGRADE_INTERVAL", 7*24*time.Hour),
		StaleLeadAge:                      envDuration("PARSER_STALE_LEAD_AGE", 30*24*time.Hour),
		DuplicateSuggestInterval:          envDuration("PARSER_DUPLICATE_SUGGEST_INTERVAL", 24*time.Hour),
		DuplicateSuggestWindow:            envDuration("PARSER_DUPLICATE_SUGGEST_WINDOW", 7*24*time.Hour),
		GeoAuditInterval:                  envDuration("PARSER_GEO_AUDIT_INTERVAL", 7*24*time.Hour),
		GeoAuditSampleN:                   envInt("PARSER_GEO_AUDIT_SAMPLE_N", 10),
		GeoAuditCollection:                env("PARSER_GEO_AUDIT_COLLECTION", "geo_audit_reports"),
		WebhookAuditInterval:              envDuration("PARSER_WEBHOOK_AUDIT_INTERVAL", 30*24*time.Hour),
		WebhookFeedbackCollection:         env("PARSER_WEBHOOK_FEEDBACK_COLLECTION", "webhook_feedback"),
		ParserChannelTriage:               envBool("PARSER_CHANNEL_TRIAGE", false),
		BGChannelTriageInterval:           envDuration("PARSER_BG_CHANNEL_TRIAGE_INTERVAL", 6*time.Hour),
		ParserDomainTriage:                envBool("PARSER_DOMAIN_TRIAGE", false),
		ParserLanderPathDiscover:          envBool("PARSER_LANDER_PATH_DISCOVER", true),
		ParserLanderPathGemini:            envBool("PARSER_LANDER_PATH_GEMINI", false),
		LanderPathsCachePath:              env("LANDER_PATHS_CACHE_PATH", "data/runtime/lander_paths.json"),
		BGDomainTriageInterval:            envDuration("PARSER_BG_DOMAIN_TRIAGE_INTERVAL", 6*time.Hour),
		DomainTriageCachePath:             env("DOMAIN_TRIAGE_CACHE_PATH", "data/runtime/domain_triage_cache.json"),
		TelegramChannelsPath:              env("TELEGRAM_CHANNELS_PATH", "data/runtime/discovered_telegram_channels.json"),
		GeminiEmbedPainMin:                envFloat("GEMINI_EMBED_PAIN_MIN", 0.78),
		GeminiEmbedSpamMin:                envFloat("GEMINI_EMBED_SPAM_MIN", 0.82),
		GeminiLeadAnalyzeInterval:         envDuration("GEMINI_LEAD_ANALYZE_INTERVAL", 5*time.Minute),
		GeminiLeadBatchSize:               envInt("GEMINI_LEAD_BATCH_SIZE", 15),
		GeminiQuotaCriticalPct:            envInt("GEMINI_QUOTA_CRITICAL_PCT", 20),
		GeminiQuotaHighPct:                envInt("GEMINI_QUOTA_HIGH_PCT", 40),
		GeminiQuotaNormalPct:              envInt("GEMINI_QUOTA_NORMAL_PCT", 25),
		GeminiQuotaLowPct:                 envInt("GEMINI_QUOTA_LOW_PCT", 15),
		WarmLeadQueueSize:                 envInt("WARM_LEAD_QUEUE_SIZE", 256),
		WarmAnalysisRetryMax:              envInt("WARM_ANALYSIS_RETRY_MAX", 3),
		WarmAnalysisRetryBase:             envDuration("WARM_ANALYSIS_RETRY_BASE", 5*time.Second),
		WarmAnalysisPendingScanInterval:   envDuration("WARM_ANALYSIS_PENDING_SCAN_INTERVAL", 15*time.Minute),
		WarmAnalysisPendingStale:          envDuration("WARM_ANALYSIS_PENDING_STALE", time.Hour),
		WarmAnalysisDLQCollection:         env("WARM_ANALYSIS_DLQ_COLLECTION", "warm_analysis_dlq"),
		WarmAnalysisShutdownDrain:         envDuration("WARM_ANALYSIS_SHUTDOWN_DRAIN", 2*time.Minute),
		ParserICPClassify:                 envBool("PARSER_ICP_CLASSIFY", false),
		ParserICPClassifyTgWeb:            envBool("PARSER_ICP_CLASSIFY_TGWEB", true),
		ParserTgWebPrescanMode:            env("PARSER_TGWEB_PRESCAN_MODE", "aggressive"),
		ParserGeoClassify:                 envBool("PARSER_GEO_CLASSIFY", false),
		ParserTimeDecay:                   envBool("PARSER_TIME_DECAY", true),
		EnrichRDAP:                        envBool("PARSER_ENRICH_RDAP", true),
		EnrichDNS:                         envBool("PARSER_ENRICH_DNS", true),
		EnrichEmail:                       envBool("PARSER_ENRICH_EMAIL", true),
		EnrichSMTPVerify:                  envBool("PARSER_ENRICH_SMTP_VERIFY", false),
		DiscordBotToken:                   env("DISCORD_BOT_TOKEN", ""),
		DiscordChannelIDs:                 parseCSV(env("DISCORD_CHANNEL_IDS", "")),
		DiscordMaxMessages:                envInt("DISCORD_MAX_MESSAGES", 50),
		DiscordRegistryPath:               env("DISCORD_REGISTRY_PATH", "data/runtime/discovered_discord_invites.json"),
		SourceStatsCollection:             env("SOURCE_STATS_COLLECTION", "source_stats"),
		CrmBoostCollection:                env("CRM_BOOST_COLLECTION", "crm_boosts"),
		EmbeddingCollection:               env("EMBEDDING_COLLECTION", "snippet_embeddings"),
		BlacklistDomainsPath:              env("BLACKLIST_DOMAINS_PATH", "data/blacklist_domains.txt"),
		BlacklistEmailsPath:               env("BLACKLIST_EMAILS_PATH", "data/blacklist_emails.txt"),
		ParserPilotTag:                    envBool("PARSER_PILOT_TAG", true),
		ParserGeminiEngage:                envBool("PARSER_GEMINI_ENGAGE", false),
		ParserEntityOutreachNarrative:     envBool("PARSER_ENTITY_OUTREACH_NARRATIVE", false),
		ParserEmbedPrescan:                envBool("PARSER_EMBED_PRESCAN", false),
		ParserWarmEmbedPrescan:            envBool("PARSER_WARM_EMBED_PRESCAN", false),
		ParserEmbedCluster:                envBool("PARSER_EMBED_CLUSTER", false),
		ParserWarmEmbedCluster:            envBool("PARSER_WARM_EMBED_CLUSTER", false),
		ParserGeminiEngageMedium:          envBool("PARSER_GEMINI_ENGAGE_MEDIUM", false),
		ParserGeminiEnrichSynth:           envBool("PARSER_GEMINI_ENRICH_SYNTH", false),
		ParserGeminiDefer:                 envBool("PARSER_GEMINI_DEFER", true),
		ParserGeminiSyncGeo:               envBool("PARSER_GEMINI_SYNC_GEO", false),
		ParserLeadStatusEnabled:           envBool("PARSER_LEAD_STATUS_ENABLED", false),
		KeywordsLocalePath:                env("KEYWORDS_LOCALE_PATH", ""),
		KeywordsLocale:                    env("KEYWORDS_LOCALE", ""),
		CTQueries:                         parseCSV(env("CT_QUERIES", "track,click,go")),
		CTMaxResults:                      envInt("CT_MAX_RESULTS", 100),
		GitHubToken:                       env("GITHUB_TOKEN", ""),
		GitHubSearchQueries:               parseSemicolonCSV(env("GITHUB_SEARCH_QUERIES", "voluum alternative;self-hosted tracker;keitaro docker")),
		GitHubRotateEnabled:               envBool("PARSER_GITHUB_ROTATE", false),
		GitHubRotateStatePath:             env("GITHUB_ROTATE_STATE_PATH", "data/runtime/github_query_rotate.json"),
		MetricsAddr:                       env("PARSER_METRICS_ADDR", ""),
		CRMWebhookURL:                     env("PARSER_CRM_WEBHOOK_URL", ""),
		CRMWebhookSecret:                  env("PARSER_CRM_WEBHOOK_SECRET", ""),
		CRMWebhookEnabled:                 envBool("PARSER_CRM_WEBHOOK", false),
		CRMWebhookAfterAnalysis:           envBool("PARSER_CRM_WEBHOOK_AFTER_ANALYSIS", false),
		CRMWebhookHeatMin:                 env("PARSER_CRM_WEBHOOK_HEAT_MIN", ""),
		TelegramProxyURL:                  env("TELEGRAM_PROXY_URL", ""),
		BGWorkerEnabled:                   envBool("PARSER_BG_WORKER", true),
		BGTelegramEnabled:                 envBool("PARSER_BG_TELEGRAM", true),
		BGSerpTelegramInterval:            time.Duration(envInt("PARSER_BG_SERP_TELEGRAM_MIN", 60)) * time.Minute,
		BGTelegramDiscoverInterval:        time.Duration(envInt("PARSER_BG_TELEGRAM_DISCOVER_MIN", 360)) * time.Minute,
		BGTelegramScrapeInterval:          time.Duration(envInt("PARSER_BG_TELEGRAM_SCRAPE_MIN", 30)) * time.Minute,
		BGTelegramWebInterval:             time.Duration(envInt("PARSER_BG_TELEGRAM_WEB_MIN", 120)) * time.Minute,
		BGForumDiscoverInterval:           envDuration("PARSER_BG_FORUM_DISCOVER_INTERVAL", 12*time.Hour),
		BGSourceRegistrySyncInterval:      time.Duration(envInt("PARSER_BG_SOURCE_REGISTRY_SYNC_MIN", 30)) * time.Minute,
		BGAutoReportInterval:              envDuration("PARSER_BG_AUTO_REPORT_INTERVAL", 7*24*time.Hour),
		BGDiscordDiscoverInterval:         envDuration("PARSER_BG_DISCORD_DISCOVER_INTERVAL", 24*time.Hour),
		BGSourceDisableInterval:           envDuration("PARSER_BG_SOURCE_DISABLE_INTERVAL", 24*time.Hour),
		AutoReportPath:                    env("PARSER_AUTO_REPORT_PATH", "data/runtime/auto_report.jsonl"),
		DisabledSourcesPath:               env("PARSER_DISABLED_SOURCES_PATH", "data/runtime/disabled_sources.json"),
		SourceDisableMinRaw:               envInt("PARSER_SOURCE_DISABLE_MIN_RAW", 100),
		ParserSourceDisableGovernor:       envBool("PARSER_SOURCE_DISABLE_GOVERNOR", false),
		ParserAutoDiscover:                envBool("PARSER_AUTO_DISCOVER", false),
		ParserSeedFeedback:                envBool("PARSER_SEED_FEEDBACK", false),
		ParserSeedFeedbackMinHeat:         env("PARSER_SEED_FEEDBACK_MIN_HEAT", "hot"),
		DiscoverAutoApplyMaxWeek:          envInt("PARSER_DISCOVER_AUTO_APPLY_MAX_WEEK", 30),
		TelegramDomainsPath:               env("TELEGRAM_DOMAINS_PATH", "data/runtime/discovered_telegram_domains.json"),
		TelegramWebMaxDomains:             envInt("TELEGRAM_WEB_MAX_DOMAINS", 25),
		TelegramWebRescanDays:             envInt("TELEGRAM_WEB_RESCAN_DAYS", 30),
		TelegramWebDomains:                parseCSV(env("TELEGRAM_WEB_DOMAINS", "")),
		ProcessorTaskTimeout:              envDuration("PARSER_TASK_TIMEOUT", 90*time.Second),
		WarriorSeedPath:                   env("WARRIOR_SEED_PATH", "data/seeds/warrior_threads.csv"),
		WarriorHostRPS:                    envFloat("WARRIOR_HOST_RPS", 1),
		KeywordStatsCollection:            env("KEYWORD_STATS_COLLECTION", "keyword_stats"),
		ParserEntitySightings:             envBool("PARSER_ENTITY_SIGHTINGS", true),
		EntityCollection:                  env("ENTITY_COLLECTION", "entities"),
		ParserCrossSourceHot:              envBool("PARSER_CROSS_SOURCE_HOT", true),
		CrossSourceHotWindow:              envDuration("PARSER_CROSS_SOURCE_HOT_WINDOW", 7*24*time.Hour),
		CrossSourceHotBoost:               envInt("PARSER_CROSS_SOURCE_HOT_BOOST", 20),
		ParserEntityHeatEnabled:           envBool("PARSER_ENTITY_HEAT_ENABLED", true),
		EntityHeatBlazing:                 envFloat("PARSER_ENTITY_HEAT_BLAZING", 80),
		EntityHeatHot:                     envFloat("PARSER_ENTITY_HEAT_HOT", 50),
		EntityHeatWarm:                    envFloat("PARSER_ENTITY_HEAT_WARM", 25),
		EntityHeatDecay7D:                 envFloat("PARSER_ENTITY_HEAT_DECAY_7D", 1.0),
		EntityHeatDecay30D:                envFloat("PARSER_ENTITY_HEAT_DECAY_30D", 0.6),
		EntityHeatDecay90D:                envFloat("PARSER_ENTITY_HEAT_DECAY_90D", 0.25),
		ParserEntityGeminiEnabled:         envBool("PARSER_ENTITY_GEMINI_ENABLED", true),
		EntityGeminiDebounce:              envDuration("PARSER_ENTITY_GEMINI_DEBOUNCE", 6*time.Hour),
		EntityGeminiLowConfidenceDebounce: envDuration("PARSER_ENTITY_GEMINI_LOW_CONF_DEBOUNCE", time.Hour),
		EntityGeminiInterval:              envDuration("PARSER_ENTITY_GEMINI_INTERVAL", 5*time.Minute),
		EntityGeminiQueueSize:             envInt("PARSER_ENTITY_GEMINI_QUEUE_SIZE", 128),
		ParserEntityLinkSuggest:           envBool("PARSER_ENTITY_LINK_SUGGEST", false),
		EntityLinkSuggestInterval:         envDuration("PARSER_ENTITY_LINK_SUGGEST_INTERVAL", 6*time.Hour),
	}

	apiID, err := envIntOptional("TELEGRAM_API_ID")
	if err != nil {
		return Config{}, fmt.Errorf("TELEGRAM_API_ID: %w", err)
	}
	cfg.TelegramAPIID = apiID

	applyComplianceDefaults(&cfg)

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
		// Invalid env value: keep default rather than fail startup (typo-tolerant ops).
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
		return fallback // same typo-tolerant policy as envInt
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
