package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	parsercfg "github.com/bidshard/parser/internal/config"
)

type Config struct {
	MongoURI               string
	MongoDB                string
	MongoCollection        string
	SourceStatsCollection  string
	KeywordStatsCollection string
	CrmBoostCollection     string
	LeadNotesCollection    string
	LeadCrmMetaCollection  string
	EntityCollection       string
	ShutdownTimeout        time.Duration
	QueryTimeout           time.Duration
	WriteTimeout           time.Duration
	StatsTimeout           time.Duration
	WebhookAddr            string
	WebhookSecret          string
	MetricsAddr            string
	PprofAddr              string
	ExportMaxRows          int
	SearchTimeout          time.Duration
	SearchMaxRows          int
	LogFormat              string
	LogLevel               string
}

func Load() (Config, error) {
	parsercfg.LoadDotEnv()

	cfg := Config{
		MongoURI:               env("MONGO_URI", ""),
		MongoDB:                env("MONGO_DB", "parser"),
		MongoCollection:        env("PARSER_MONGO_COLLECTION", "leads"),
		SourceStatsCollection:  env("SOURCE_STATS_COLLECTION", "source_stats"),
		KeywordStatsCollection: env("KEYWORD_STATS_COLLECTION", "keyword_stats"),
		CrmBoostCollection:     env("CRM_BOOST_COLLECTION", "crm_boosts"),
		LeadNotesCollection:    env("CRM_LEAD_NOTES_COLLECTION", "lead_notes"),
		LeadCrmMetaCollection:  env("CRM_META_COLLECTION", "lead_crm_meta"),
		EntityCollection:       env("ENTITY_COLLECTION", "entities"),
		ShutdownTimeout:        envDuration("CRM_SHUTDOWN_TIMEOUT", 30*time.Second),
		QueryTimeout:           envDuration("CRM_QUERY_TIMEOUT", 5*time.Second),
		WriteTimeout:           envDuration("CRM_WRITE_TIMEOUT", 3*time.Second),
		StatsTimeout:           envDuration("CRM_STATS_TIMEOUT", 15*time.Second),
		WebhookAddr:            env("CRM_WEBHOOK_ADDR", "127.0.0.1:8080"),
		WebhookSecret:          strings.TrimSpace(env("CRM_WEBHOOK_SECRET", "")),
		MetricsAddr:            env("CRM_METRICS_ADDR", ""),
		PprofAddr:              env("CRM_PPROF_ADDR", ""),
		ExportMaxRows:          envInt("CRM_EXPORT_MAX_ROWS", 500),
		SearchTimeout:          envDuration("CRM_SEARCH_TIMEOUT", 5*time.Second),
		SearchMaxRows:          envInt("CRM_SEARCH_MAX_ROWS", 20),
		LogFormat:              env("CRM_LOG_FORMAT", "auto"),
		LogLevel:               env("CRM_LOG_LEVEL", "info"),
	}
	return cfg, nil
}

func (c Config) ValidateForRun() []string {
	var errs []string
	if strings.TrimSpace(c.MongoURI) == "" {
		errs = append(errs, "MONGO_URI empty")
	}
	if strings.TrimSpace(c.WebhookAddr) == "" {
		errs = append(errs, "CRM_WEBHOOK_ADDR empty")
	}
	return errs
}

type ConfigView struct {
	MongoURI               string `json:"mongo_uri"`
	MongoDB                string `json:"mongo_db"`
	MongoCollection        string `json:"mongo_collection"`
	SourceStatsCollection  string `json:"source_stats_collection"`
	KeywordStatsCollection string `json:"keyword_stats_collection"`
	CrmBoostCollection     string `json:"crm_boost_collection"`
	LeadNotesCollection    string `json:"lead_notes_collection"`
	LeadCrmMetaCollection  string `json:"lead_crm_meta_collection"`
	ShutdownTimeout        string `json:"shutdown_timeout"`
	QueryTimeout           string `json:"query_timeout"`
	WriteTimeout           string `json:"write_timeout"`
	StatsTimeout           string `json:"stats_timeout"`
	WebhookAddr            string `json:"webhook_addr"`
	WebhookSecret          string `json:"webhook_secret"`
	MetricsAddr            string `json:"metrics_addr"`
	PprofAddr              string `json:"pprof_addr"`
	ExportMaxRows          int    `json:"export_max_rows"`
	SearchTimeout          string `json:"search_timeout"`
	SearchMaxRows          int    `json:"search_max_rows"`
	LogFormat              string `json:"log_format"`
	LogLevel               string `json:"log_level"`
}

func (c Config) View() ConfigView {
	return ConfigView{
		MongoURI:               c.MongoURI,
		MongoDB:                c.MongoDB,
		MongoCollection:        c.MongoCollection,
		SourceStatsCollection:  c.SourceStatsCollection,
		KeywordStatsCollection: c.KeywordStatsCollection,
		CrmBoostCollection:     c.CrmBoostCollection,
		LeadNotesCollection:    c.LeadNotesCollection,
		LeadCrmMetaCollection:  c.LeadCrmMetaCollection,
		ShutdownTimeout:        c.ShutdownTimeout.String(),
		QueryTimeout:           c.QueryTimeout.String(),
		WriteTimeout:           c.WriteTimeout.String(),
		StatsTimeout:           c.StatsTimeout.String(),
		WebhookAddr:            c.WebhookAddr,
		WebhookSecret:          maskSecret(c.WebhookSecret),
		MetricsAddr:            c.MetricsAddr,
		PprofAddr:              c.PprofAddr,
		ExportMaxRows:          c.ExportMaxRows,
		SearchTimeout:          c.SearchTimeout.String(),
		SearchMaxRows:          c.SearchMaxRows,
		LogFormat:              c.LogFormat,
		LogLevel:               c.LogLevel,
	}
}

func maskSecret(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "****"
	}
	return v[:2] + "****" + v[len(v)-2:]
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
