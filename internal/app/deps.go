package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/coldpath"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/enrich"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/ingest"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/output"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/telethon"
	"github.com/bidshard/parser/internal/validate"
	"go.mongodb.org/mongo-driver/mongo"
)

type runtimeDeps struct {
	processor   *pipeline.Processor
	bulkStore   *sink.BulkStore
	coldPath    *coldpath.Service
	mongoClient *mongo.Client
}

func buildDeps(ctx context.Context, cfg config.Config) (*runtimeDeps, error) {
	reg := scoring.NewRegistry(cfg.KeywordsJSONPath)
	overlays := []string{cfg.KeywordsGrayPath}
	if cfg.KeywordsLocalePath != "" {
		overlays = append(overlays, cfg.KeywordsLocalePath)
	} else if cfg.KeywordsLocale != "" {
		overlays = append(overlays, "data/keywords-"+cfg.KeywordsLocale+".json")
	}
	if err := reg.LoadWithOverlays(ctx, cfg.KeywordsJSONPath, overlays...); err != nil {
		return nil, err
	}
	if err := validate.LoadDisposableDomains(cfg.DisposableDomainsPath); err != nil {
		slog.Warn("disposable domains load failed", "path", cfg.DisposableDomainsPath, "error", err)
	} else if cfg.DisposableDomainsPath != "" {
		slog.Info("disposable domains loaded", "path", cfg.DisposableDomainsPath, "count", validate.DisposableDomainCount())
	}
	if err := validate.LoadBlacklistDomains(cfg.BlacklistDomainsPath); err != nil {
		slog.Warn("blacklist domains load failed", "path", cfg.BlacklistDomainsPath, "error", err)
	} else if cfg.BlacklistDomainsPath != "" {
		slog.Info("blacklist domains loaded", "path", cfg.BlacklistDomainsPath, "count", validate.BlacklistDomainCount())
	}
	if err := validate.LoadBlacklistEmails(cfg.BlacklistEmailsPath); err != nil {
		slog.Warn("blacklist emails load failed", "path", cfg.BlacklistEmailsPath, "error", err)
	} else if cfg.BlacklistEmailsPath != "" {
		slog.Info("blacklist emails loaded", "path", cfg.BlacklistEmailsPath, "count", validate.BlacklistEmailCount())
	}

	_ = httpclient.Shared(cfg.HTTPTimeout)

	mx := validate.MXValidator(validate.StubMX{OK: true})
	if cfg.MXCheck {
		mx = validate.Resolver{}
	}

	deps := &runtimeDeps{}

	var junkCapturer *coldpath.Capturer
	var sourceStats *sink.SourceStatsStore
	var keywordStats *sink.KeywordStatsStore
	var geminiClient *gemini.Client

	if cfg.GeminiAPIKey != "" {
		client, err := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel, gemini.ClientOptionsFrom(cfg)...)
		if err != nil {
			slog.Warn("gemini client init failed", "error", err)
		} else {
			geminiClient = client
			limits := client.Limits()
			slog.Info("gemini quotas configured",
				"model", client.Model(),
				"rpm", limits.RPM,
				"tpm", limits.TPM,
				"rpd", limits.RPD,
				"max_retries", limits.MaxRetries,
			)
		}
	}

	if cfg.MongoURI != "" {
		client, err := sink.ConnectMongoClient(ctx, cfg.MongoURI)
		if err != nil {
			slog.Warn("mongo connect failed", "error", err)
		} else {
			deps.mongoClient = client
			if stats, err := sink.ConnectSourceStats(ctx, client, cfg.MongoDB, cfg.SourceStatsCollection); err != nil {
				slog.Warn("source stats store failed", "error", err)
			} else {
				sourceStats = stats
			}
			if stats, err := sink.ConnectKeywordStats(ctx, client, cfg.MongoDB, cfg.KeywordStatsCollection); err != nil {
				slog.Warn("keyword stats store failed", "error", err)
			} else {
				keywordStats = stats
				slog.Info("keyword stats store connected", "collection", cfg.KeywordStatsCollection)
			}
		}
	}

	sourceRep := scoring.NewSourceReputation(sourceStats)

	if cfg.GeminiAPIKey != "" && cfg.MongoURI != "" && deps.mongoClient != nil && geminiClient != nil {
		junkCapturer = coldpath.NewCapturer(cfg.ColdJunkQueueSize)
		junkStore, err := sink.ConnectJunkStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.ColdJunkCollection, cfg.ColdReportCollection)
		if err != nil {
			slog.Warn("cold path junk store failed", "error", err)
		} else {
			crmStore, _ := sink.ConnectCrmBoost(ctx, deps.mongoClient, cfg.MongoDB, cfg.CrmBoostCollection)
			embedStore, _ := sink.ConnectEmbeddingStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.EmbeddingCollection)
			deps.coldPath = coldpath.NewService(coldpath.Config{
				AnalyzeInterval:  cfg.GeminiAnalyzeInterval,
				ReportInterval:   cfg.GeminiReportInterval,
				BatchSize:        cfg.GeminiBatchSize,
				KeywordDiffEvery: cfg.GeminiKeywordDiffEvery,
				KeywordDiffDir:   cfg.GeminiKeywordDiffDir,
				EmbedThreshold:   cfg.GeminiEmbedThreshold,
			}, junkCapturer, junkStore, geminiClient, crmStore, embedStore, keywordStats)
			slog.Info("cold path gemini enabled",
				"model", cfg.GeminiModel,
				"analyze_interval", cfg.GeminiAnalyzeInterval,
				"report_interval", cfg.GeminiReportInterval,
				"keyword_diff_every", cfg.GeminiKeywordDiffEvery,
				"embed_threshold", cfg.GeminiEmbedThreshold,
			)
		}
	}

	inner := sink.OpenStore(ctx, cfg.MongoURI, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots, cfg.ExportJSONPath)
	bulk := sink.NewBulkStore(inner, 50, 2*time.Second)

	httpClient := httpclient.Shared(cfg.HTTPTimeout)
	var enricher *enrich.Enricher
	if cfg.EnrichRDAP || cfg.EnrichDNS || cfg.EnrichEmail {
		enricher = enrich.New(enrich.Config{
			BlockedCountries: cfg.GeoBlockCountries,
			RDAPEnabled:      cfg.EnrichRDAP,
			DNSEnabled:       cfg.EnrichDNS,
			EmailEnabled:     cfg.EnrichEmail,
			SMTPVerify:       cfg.EnrichSMTPVerify,
		}, enrich.NewRDAPLookup(httpClient), enrich.NewDNSLookup(), enrich.NewEmailLookup(httpClient, cfg.EnrichSMTPVerify, cfg.HTTPTimeout))
	}

	deps.processor = &pipeline.Processor{
		Registry:          reg,
		Seen:              dedup.NewSeenCache(50_000, 24*time.Hour),
		Store:             bulk,
		MX:                mx,
		Junk:              junkCapturer,
		SourceRep:         sourceRep,
		KeywordStats:      keywordStats,
		ICP:               geminiClient,
		ICPEnabled:        cfg.ParserICPClassify && geminiClient != nil,
		Geo:               geminiClient,
		GeoEnabled:        cfg.ParserGeoClassify && geminiClient != nil,
		GeoBlockCountries: cfg.GeoBlockCountries,
		Enricher:          enricher,
		TimeDecayEnabled:  cfg.ParserTimeDecay,
		PilotTagEnabled:   cfg.ParserPilotTag,
		LeadStatusEnabled: cfg.ParserLeadStatusEnabled,
	}
	deps.bulkStore = bulk

	if cfg.MetricsAddr != "" {
		metrics.StartMetricsServer(ctx, cfg.MetricsAddr)
		slog.Info("prometheus metrics endpoint enabled", "addr", cfg.MetricsAddr)
	}

	return deps, nil
}

func (d *runtimeDeps) flushStore(ctx context.Context) {
	if d == nil || d.bulkStore == nil {
		return
	}
	if err := d.bulkStore.Flush(ctx); err != nil {
		slog.Warn("bulk store flush failed", "error", err)
	}
}

func (d *runtimeDeps) closeMongo(ctx context.Context) {
	if d == nil || d.mongoClient == nil {
		return
	}
	if err := d.mongoClient.Disconnect(ctx); err != nil {
		slog.Warn("mongo disconnect failed", "error", err)
	}
}

func runIngestOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps, reader io.Reader) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan pipeline.Task, cfg.TaskBuffer)
	statsCh := make(chan pipeline.RoundStats, 8)

	var wg sync.WaitGroup
	onceDone := make(chan struct{}, 1)

	if deps.coldPath != nil {
		deps.coldPath.Run(ctx, &wg)
	}

	pool := pipeline.NewPool(cfg.WorkerCount, deps.processor)
	pool.Run(ctx, &wg, taskCh)

	reporter := output.NewReporter(cfg.Output, nil)
	reporter.SetOnReport(func() {
		select {
		case onceDone <- struct{}{}:
		default:
		}
	})
	reporter.Run(ctx, &wg, statsCh)

	roundID := newRoundID()
	state := &pipeline.RoundState{}
	start := time.Now()

	ingest.Scan(ctx, reader, taskCh, state, roundID)
	state.Wait()
	stats := state.Snapshot(roundID, time.Since(start))
	select {
	case statsCh <- stats:
	default:
	}

	slog.Info("scan round finished",
		"round_id", stats.RoundID,
		"duration_ms", stats.Duration.Milliseconds(),
		"sources_ok", stats.SourcesOK,
		"sources_fail", stats.SourcesFail,
		"raw", stats.RawTotal,
		"accepted", stats.Accepted,
		"rejected_geo", stats.RejectedGeo,
		"dropped", stats.Dropped,
		"high", stats.High,
		"medium", stats.Medium,
	)

	select {
	case <-onceDone:
	case <-time.After(2 * time.Second):
	}

	cancel()
	Drain(cancel, &wg, taskCh, cfg.ShutdownTimeout)
	return nil
}

func newRoundID() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func runTelegramSidecarOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps) error {
	pr, pw := io.Pipe()

	sidecarCtx, sidecarCancel := context.WithCancel(ctx)
	defer sidecarCancel()

	errCh := make(chan error, 1)
	go func() {
		err := telethon.Run(sidecarCtx, telethon.Options{
			ConfigPath: cfg.TelegramConfigPath,
			DryRun:     cfg.TelegramDryRun,
			Once:       true,
		}, pw)
		_ = pw.Close()
		errCh <- err
	}()

	ingestErr := runIngestOnce(ctx, cfg, deps, pr)
	sidecarErr := <-errCh

	if ingestErr != nil {
		return ingestErr
	}
	if sidecarErr != nil && sidecarErr != context.Canceled {
		return sidecarErr
	}
	return nil
}
