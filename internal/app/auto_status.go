package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/lander"
	"github.com/bidshard/parser/internal/sources/tgweb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AutoStatus is a point-in-time automation health snapshot (no PII).
type AutoStatus struct {
	At                  time.Time              `json:"at"`
	AutoDiscover        bool                   `json:"auto_discover"`
	ForumThreads        int                    `json:"forum_threads"`
	TelegramDomains     int                    `json:"telegram_domains"`
	SourceRegistry      map[string]int         `json:"source_registry"`
	RegistryDropped     int                    `json:"registry_dropped"`
	HeadlessQueued      int                    `json:"headless_queued"`
	ProxyUsedMB         float64                `json:"proxy_used_mb"`
	ProxyCapMB          int                    `json:"proxy_cap_mb"`
	ProxyBudgetExceeded bool                   `json:"proxy_budget_exceeded"`
	AnalysisPending     int64                  `json:"analysis_pending"`
	MongoOK             bool                   `json:"mongo_ok"`
	Egress              metrics.EgressCounters `json:"egress"`
}

// CollectAutoStatus reads runtime registries and optional Mongo counters.
func CollectAutoStatus(ctx context.Context, cfg config.Config) AutoStatus {
	st := AutoStatus{
		At:              time.Now().UTC(),
		AutoDiscover:    cfg.ParserAutoDiscover,
		SourceRegistry:  map[string]int{},
		AnalysisPending: -1,
	}
	if reg, err := forum.LoadThreadRegistry(cfg.ForumRegistryPath); err == nil {
		st.ForumThreads = len(reg.Threads)
	}
	if f, err := tgweb.LoadDomains(cfg.TelegramDomainsPath); err == nil {
		st.TelegramDomains = len(f.Domains)
	}
	if f, err := sourceregistry.Load(cfg.SourceRegistryPath); err == nil {
		for _, e := range f.Sources {
			for _, typ := range e.Types {
				st.SourceRegistry[strings.ToLower(typ)]++
			}
			if strings.EqualFold(strings.TrimSpace(e.TriageStatus), "drop") {
				st.RegistryDropped++
			}
		}
	}
	if n, err := lander.CountHeadlessQueue(cfg.LanderHeadlessQueuePath); err == nil {
		st.HeadlessQueued = n
	}
	st.ProxyCapMB = cfg.ProxyDailyMBCap
	if cfg.ProxyDailyMBCap > 0 {
		used, exceeded := readProxyBudgetState(cfg.ProxyBudgetStatePath, cfg.ProxyDailyMBCap)
		st.ProxyUsedMB = float64(used) / (1024 * 1024)
		st.ProxyBudgetExceeded = exceeded
	} else if g := proxybudget.Current(); g != nil && g.Enabled() {
		st.ProxyCapMB = int(g.CapBytes() / (1024 * 1024))
		st.ProxyUsedMB = float64(g.UsedBytes()) / (1024 * 1024)
	}
	if strings.TrimSpace(cfg.MongoURI) != "" {
		if n, ok := countPendingAnalysis(ctx, cfg); ok {
			st.AnalysisPending = n
			st.MongoOK = true
		}
	}
	st.Egress = metrics.SnapshotEgress()
	return st
}

func readProxyBudgetState(path string, capMB int) (used int64, exceeded bool) {
	if path == "" {
		path = proxybudget.DefaultStatePath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	var st struct {
		Day   string `json:"day"`
		Bytes int64  `json:"bytes"`
	}
	if json.Unmarshal(raw, &st) != nil {
		return 0, false
	}
	if st.Day != time.Now().UTC().Format("2006-01-02") {
		return 0, false
	}
	capBytes := int64(capMB) * 1024 * 1024
	return st.Bytes, capBytes > 0 && st.Bytes >= capBytes
}

func countPendingAnalysis(ctx context.Context, cfg config.Config) (int64, bool) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return 0, false
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	store, err := sink.NewMongoStoreFromClient(ctx, client, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots)
	if err != nil {
		return 0, false
	}
	n, err := store.CountPendingAnalysis(ctx)
	if err != nil {
		return 0, false
	}
	return n, true
}

// WriteAutoStatus prints human-readable automation health (F-03).
func WriteAutoStatus(w io.Writer, st AutoStatus) {
	_, _ = fmt.Fprintf(w, "auto status at %s\n", st.At.Format(time.RFC3339))
	if st.AutoDiscover {
		_, _ = fmt.Fprintln(w, "ok  PARSER_AUTO_DISCOVER=true")
	}
	_, _ = fmt.Fprintf(w, "forum_threads=%d telegram_domains=%d registry_dropped=%d headless_queue=%d\n",
		st.ForumThreads, st.TelegramDomains, st.RegistryDropped, st.HeadlessQueued)
	if len(st.SourceRegistry) > 0 {
		_, _ = fmt.Fprintf(w, "source_registry tgweb=%d supply=%d lander=%d\n",
			st.SourceRegistry[sourceregistry.TypeTGWeb],
			st.SourceRegistry[sourceregistry.TypeSupply],
			st.SourceRegistry[sourceregistry.TypeLander])
	}
	if st.ProxyCapMB > 0 {
		_, _ = fmt.Fprintf(w, "proxy_budget=%.1f/%d MB exceeded=%v\n",
			st.ProxyUsedMB, st.ProxyCapMB, st.ProxyBudgetExceeded)
	}
	if st.AnalysisPending >= 0 {
		_, _ = fmt.Fprintf(w, "analysis_pending=%d\n", st.AnalysisPending)
	} else if st.MongoOK {
		_, _ = fmt.Fprintln(w, "analysis_pending=0")
	}
	if st.Egress.ProxyCfBlock > 0 || st.Egress.ProxyCooldownWait > 0 || st.Egress.ProxyTransportFail > 0 ||
		st.Egress.CrawlHTTPFail > 0 || st.Egress.SerpHarvestFailed > 0 || st.Egress.GeminiJunkBatchFailed > 0 ||
		st.Egress.TelethonSidecarFailed > 0 {
		_, _ = fmt.Fprintf(w, "egress proxy_cf_block=%d proxy_cooldown_wait=%d proxy_transport_fail=%d crawl_http_fail=%d serp_fail=%d gemini_junk_fail=%d telethon_fail=%d\n",
			st.Egress.ProxyCfBlock,
			st.Egress.ProxyCooldownWait,
			st.Egress.ProxyTransportFail,
			st.Egress.CrawlHTTPFail,
			st.Egress.SerpHarvestFailed,
			st.Egress.GeminiJunkBatchFailed,
			st.Egress.TelethonSidecarFailed,
		)
	}
}

// WriteAutoReportJSONL appends one JSON line snapshot (F-02, no PII).
func WriteAutoReportJSONL(path string, st AutoStatus) error {
	if path == "" {
		path = "data/runtime/auto_report.jsonl"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	return err
}
