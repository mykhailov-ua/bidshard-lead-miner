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
	PollInterval       time.Duration
	WorkerCount        int
	TaskBuffer         int
	SourceConcurrency  int
	ScanTimeout        time.Duration
	HTTPTimeout        time.Duration
	ShutdownTimeout    time.Duration
	LogFormat          string
	LogLevel           string
	Output             string
	WriteSlots         int
	ScanOnce           bool
	IngestStdin        bool
	IngestReader       io.Reader
	TelegramSidecar    bool
	TelegramDryRun     bool
	TelegramConfigPath string
	Source             string
	SupplySeedPath     string
	SupplyHostRPS      float64
	SupplyMaxDomains   int
	SupplyBaseURL      string
	ForumSeedPath      string
	ForumBaseURL       string
	LanderSeedPath     string
	LanderBaseURL      string
	LanderHeadless     bool
	TelegramAPIID      int
	TelegramAPIHash    string
	MongoURI           string
	MongoDB            string
	MongoCollection    string
	ExportJSONPath     string
	MXCheck            bool
	KeywordsJSONPath   string
	KeywordsGrayPath   string
	GeoBlockCountries  []string
}

func Load() (Config, error) {
	cfg := Config{
		PollInterval:       time.Duration(envInt("PARSER_POLL_SEC", 120)) * time.Second,
		WorkerCount:        envInt("PARSER_WORKERS", 4),
		TaskBuffer:         envInt("PARSER_TASK_BUFFER", 128),
		SourceConcurrency:  envInt("PARSER_SOURCE_CONCURRENCY", 3),
		ScanTimeout:        envDuration("PARSER_SCAN_TIMEOUT", 5*time.Minute),
		HTTPTimeout:        envDuration("PARSER_HTTP_TIMEOUT", 30*time.Second),
		ShutdownTimeout:    envDuration("PARSER_SHUTDOWN_TIMEOUT", 30*time.Second),
		LogFormat:          env("PARSER_LOG_FORMAT", "json"),
		LogLevel:           env("PARSER_LOG_LEVEL", "info"),
		Output:             env("PARSER_OUTPUT", "table"),
		WriteSlots:         envInt("PARSER_WRITE_SLOTS", 8),
		TelegramAPIHash:    env("TELEGRAM_API_HASH", ""),
		MongoURI:           env("MONGO_URI", ""),
		MongoDB:            env("MONGO_DB", "parser"),
		MongoCollection:    env("PARSER_MONGO_COLLECTION", "leads"),
		ExportJSONPath:     env("PARSER_EXPORT_JSON", ""),
		MXCheck:            envBool("PARSER_MX_CHECK", false),
		KeywordsJSONPath:   env("KEYWORDS_JSON_PATH", "data/keywords.json"),
		KeywordsGrayPath:   env("KEYWORDS_GRAY_JSON_PATH", "data/keywords-gray.json"),
		GeoBlockCountries:  parseCSV(env("GEO_BLOCK_COUNTRIES", "RU,BY")),
		TelegramConfigPath: env("TELEGRAM_CONFIG_PATH", "config/sources.telegram.yaml"),
		Source:             env("PARSER_SOURCE", "stub"),
		SupplySeedPath:     env("SUPPLY_SEED_PATH", "data/seeds/domains.csv"),
		SupplyHostRPS:      envFloat("SUPPLY_HOST_RPS", 2),
		SupplyBaseURL:      env("SUPPLY_BASE_URL", ""),
		ForumSeedPath:      env("FORUM_SEED_PATH", "data/seeds/forum_threads.csv"),
		ForumBaseURL:       env("FORUM_BASE_URL", ""),
		LanderSeedPath:     env("LANDER_SEED_PATH", "data/seeds/lander_urls.csv"),
		LanderBaseURL:      env("LANDER_BASE_URL", ""),
		LanderHeadless:     envBool("PARSER_LANDER_HEADLESS", false),
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
