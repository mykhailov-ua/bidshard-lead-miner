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
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/ingest"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/output"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/seedfeedback"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/telethon"
	"github.com/bidshard/parser/internal/validate"
	"github.com/bidshard/parser/internal/warmpath"
	"go.mongodb.org/mongo-driver/mongo"
)

type runtimeDeps struct {
	processor         *pipeline.Processor
	bulkStore         *sink.BulkStore
	coldPath          *coldpath.Service
	warmPath          *warmpath.Service
	entityClassify    *warmpath.EntityClassifyService
	entityLinkSuggest *warmpath.EntityLinkSuggestService
	mongoClient       *mongo.Client
	sourceStats       *sink.SourceStatsStore
}

func buildDeps(ctx context.Context, cfg config.Config) (*runtimeDeps, error) {
	reg := scoring.NewRegistry(cfg.KeywordsJSONPath)
	overlays := []string{cfg.KeywordsGrayPath}
	overlays = append(overlays, config.KeywordOverlayPaths(cfg.KeywordsLocale, cfg.KeywordsLocalePath)...)
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

	_ = httpclient.Shared(cfg.HTTPTimeout) // warm default client for supply/enrich; HTTP crawlers use NewClientWithProxies when configured
	gov := proxybudget.Configure(cfg.ProxyDailyMBCap, cfg.ProxyBudgetStatePath)
	httpclient.SetProxyBudget(gov)
	if gov.Enabled() {
		slog.Info("proxy daily budget enabled", "cap_mb", cfg.ProxyDailyMBCap, "state", cfg.ProxyBudgetStatePath)
	}

	mx := validate.MXValidator(validate.StubMX{OK: true})
	if cfg.MXCheck {
		mx = validate.Resolver{}
	}

	deps := &runtimeDeps{}

	var junkCapturer *coldpath.Capturer
	var sourceStats *sink.SourceStatsStore
	var keywordStats *sink.KeywordStatsStore
	var entityRecorder entity.Recorder
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
				"quota_critical_pct", cfg.GeminiQuotaCriticalPct,
				"quota_high_pct", cfg.GeminiQuotaHighPct,
				"quota_normal_pct", cfg.GeminiQuotaNormalPct,
				"quota_low_pct", cfg.GeminiQuotaLowPct,
			)
		}
	}

	if cfg.MongoURI != "" {
		client, err := sink.ConnectMongoClient(ctx, cfg.MongoURI)
		if err != nil {
			slog.Warn("mongo connect failed", "error", err)
		} else {
			deps.mongoClient = client
			sourceStats = connectOptional("source stats store", func() (*sink.SourceStatsStore, error) {
				return sink.ConnectSourceStats(ctx, client, cfg.MongoDB, cfg.SourceStatsCollection)
			}, nil)
			deps.sourceStats = sourceStats
			keywordStats = connectOptional("keyword stats store", func() (*sink.KeywordStatsStore, error) {
				return sink.ConnectKeywordStats(ctx, client, cfg.MongoDB, cfg.KeywordStatsCollection)
			}, func(v *sink.KeywordStatsStore) {
				slog.Info("keyword stats store connected", "collection", cfg.KeywordStatsCollection)
			})
			if cfg.ParserEntitySightings {
				if store := connectOptional("entity store", func() (*sink.EntityStore, error) {
					return sink.ConnectEntityStore(ctx, client, cfg.MongoDB, cfg.EntityCollection)
				}, func(v *sink.EntityStore) {
					v.CrossSourceWindow = cfg.CrossSourceHotWindow
					v.HeatConfig = entityHeatFromConfig(cfg)
					slog.Info("entity sightings enabled", "collection", cfg.EntityCollection)
				}); store != nil {
					entityRecorder = store
				}
			}
		}
	}

	sourceRep := scoring.NewSourceReputation(sourceStats)

	var embedStore *sink.EmbeddingStore
	if deps.mongoClient != nil {
		embedStore = connectOptional("embedding store", func() (*sink.EmbeddingStore, error) {
			return sink.ConnectEmbeddingStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.EmbeddingCollection)
		}, nil)
	}

	if cfg.GeminiAPIKey != "" && cfg.MongoURI != "" && deps.mongoClient != nil && geminiClient != nil {
		junkCapturer = coldpath.NewCapturer(cfg.ColdJunkQueueSize)
		if junkStore := connectOptional("cold path junk store", func() (*sink.JunkStore, error) {
			return sink.ConnectJunkStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.ColdJunkCollection, cfg.ColdReportCollection)
		}, nil); junkStore != nil {
			crmStore, _ := sink.ConnectCrmBoost(ctx, deps.mongoClient, cfg.MongoDB, cfg.CrmBoostCollection)
			var geoAuditStore *sink.GeoAuditStore
			if geoStore, err := sink.ConnectGeoAuditStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.GeoAuditCollection); err == nil {
				geoAuditStore = geoStore
			}
			var boostLeads coldpath.BoostLeadLookup
			var staleRegrader *coldpath.StaleLeadRegrader
			var dupScanner *coldpath.DuplicateSuggestScanner
			var geoAudit *coldpath.GeoAuditRunner
			if leadMongo, err := sink.NewMongoStoreFromClient(ctx, deps.mongoClient, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots); err == nil {
				boostLeads = leadMongo
				if geminiClient != nil {
					staleRegrader = coldpath.NewStaleLeadRegrader(
						cfg.StaleLeadRegradeInterval,
						cfg.StaleLeadAge,
						50,
						leadMongo,
						leadMongo,
						geminiClient,
						reg,
					)
					if geoAuditStore != nil {
						geoAudit = coldpath.NewGeoAuditRunner(
							cfg.GeoAuditInterval,
							cfg.GeoAuditSampleN,
							junkStore,
							leadMongo,
							geminiClient,
							geoAuditStore,
							cfg.GeoBlockCountries,
						)
					}
				}
				if geminiClient != nil && embedStore != nil {
					dupCluster := gemini.NewLeadCluster(geminiClient, embedStore, cfg.GeminiEmbedThreshold)
					dupScanner = coldpath.NewDuplicateSuggestScanner(
						cfg.DuplicateSuggestInterval,
						cfg.DuplicateSuggestWindow,
						200,
						leadMongo,
						leadMongo,
						dupCluster,
					)
				}
			}
			var webhookAudit *coldpath.WebhookAuditReporter
			if feedbackStore := connectOptional("webhook feedback store", func() (*sink.WebhookFeedbackStore, error) {
				return sink.ConnectWebhookFeedbackStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.WebhookFeedbackCollection)
			}, nil); feedbackStore != nil {
				webhookAudit = coldpath.NewWebhookAuditReporter(cfg.WebhookAuditInterval, feedbackStore, cfg.GeminiKeywordDiffDir)
			}
			var painLister entity.PainSampleLister
			if entityRecorder != nil {
				if pl, ok := entityRecorder.(entity.PainSampleLister); ok {
					painLister = pl
				}
			}
			deps.coldPath = coldpath.NewService(coldpath.Config{
				AnalyzeInterval:          cfg.GeminiAnalyzeInterval,
				ReportInterval:           cfg.GeminiReportInterval,
				BatchSize:                cfg.GeminiBatchSize,
				KeywordDiffEvery:         cfg.GeminiKeywordDiffEvery,
				KeywordDiffDir:           cfg.GeminiKeywordDiffDir,
				DiscoverDiffEvery:        cfg.GeminiDiscoverDiffEvery,
				DiscoverDiffDir:          cfg.GeminiDiscoverDiffDir,
				PainVocabDiffEvery:       cfg.GeminiPainVocabDiffEvery,
				EmbedThreshold:           cfg.GeminiEmbedThreshold,
				HardRejectShadowDailyCap: cfg.HardRejectShadowDailyCap,
			}, junkCapturer, junkStore, geminiClient, crmStore, embedStore, keywordStats, reg, boostLeads, painLister, coldpath.ServiceExtras{
				Stale:        staleRegrader,
				DupSuggest:   dupScanner,
				GeoAudit:     geoAudit,
				WebhookAudit: webhookAudit,
				SourceStats:  sourceStats,
				ChannelsPath: cfg.TelegramChannelsPath,
			})
			slog.Info("cold path gemini enabled",
				"model", cfg.GeminiModel,
				"analyze_interval", cfg.GeminiAnalyzeInterval,
				"report_interval", cfg.GeminiReportInterval,
				"keyword_diff_every", cfg.GeminiKeywordDiffEvery,
				"embed_threshold", cfg.GeminiEmbedThreshold,
			)
		}
	}

	inner := sink.OpenStore(ctx, cfg.MongoURI, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots, cfg.ExportJSONPath, cfg.ExportJSONFormat)
	var crmWebhook *sink.WebhookClient
	if inner != nil && cfg.CRMWebhookEnabled && cfg.CRMWebhookURL != "" {
		crmWebhook = sink.NewWebhookClient(cfg.CRMWebhookURL, cfg.CRMWebhookSecret, cfg.HTTPTimeout).
			WithHeatMin(cfg.CRMWebhookHeatMin)
	}

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

	var embedPrescan *gemini.Prescan
	var leadCluster *gemini.LeadCluster
	var warmCapturer *warmpath.Capturer
	var entityClassifyCapturer *warmpath.EntityClassifyCapturer
	var leadPatcher sink.LeadAnalysisPatcher
	var deferCRMWebhook bool

	geminiDefer := cfg.ParserGeminiDefer && geminiClient != nil && deps.mongoClient != nil
	// ParserGeminiDefer matrix when true:
	//   ON:  warm-path batch (geo/ICP/engage/enrich), AnalysisStatus=pending on accept
	//   OFF: inline prescan, cluster, engage, enrich, ICP (except ICPTgWebEnabled)
	//   SYNC: geo when ParserGeminiSyncGeo; tgweb ICP always when ICPTgWebEnabled
	if geminiDefer {
		var err error
		leadPatcher, err = sink.ConnectLeadAnalysisPatcher(ctx, deps.mongoClient, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots)
		if err != nil {
			slog.Warn("warm path lead patcher failed", "error", err)
			geminiDefer = false
		} else {
			warmCapturer = warmpath.NewCapturer(cfg.WarmLeadQueueSize)
			// deferCRMWebhook set only when warm path actually starts (lead patcher connected).
			deferCRMWebhook = cfg.CRMWebhookAfterAnalysis && crmWebhook != nil
			var warmExtras warmpath.ServiceExtras
			// Second MongoStore handle on the leads collection; avoids coupling warm path to bulk upsert store.
			var leadMongo *sink.MongoStore
			if store, err := sink.NewMongoStoreFromClient(ctx, deps.mongoClient, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots); err == nil {
				leadMongo = store
				warmExtras.PendingScanner = warmpath.NewMongoPendingScanner(leadMongo)
			}
			if cfg.ExportJSONPath != "" && leadMongo != nil {
				if exportSink, err := sink.NewJSONFileSink(cfg.ExportJSONPath, cfg.ExportJSONFormat); err != nil {
					slog.Warn("warm path export sync failed", "path", cfg.ExportJSONPath, "error", err)
				} else {
					leadPatcher = sink.NewExportSyncPatcher(leadPatcher, leadMongo, exportSink)
					slog.Info("warm path export sync enabled", "path", cfg.ExportJSONPath)
				}
			}
			if cfg.WarmAnalysisDLQCollection != "" {
				if dlq, err := sink.ConnectWarmAnalysisDLQ(ctx, deps.mongoClient, cfg.MongoDB, cfg.WarmAnalysisDLQCollection, cfg.WriteSlots); err != nil {
					slog.Warn("warm path dlq connect failed", "error", err)
				} else {
					warmExtras.DLQ = warmpath.NewMongoAnalysisDLQ(dlq)
				}
			}
			if cfg.ParserWarmEmbedPrescan && geminiClient != nil {
				warmExtras.Prescan = gemini.NewPrescan(geminiClient, cfg.GeminiEmbedPainMin, cfg.GeminiEmbedSpamMin)
				if junkStore, err := sink.ConnectJunkStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.ColdJunkCollection, cfg.ColdReportCollection); err == nil {
					warmExtras.Junk = junkStore
				}
			}
			if cfg.ParserWarmEmbedCluster && geminiClient != nil && embedStore != nil {
				warmExtras.Cluster = gemini.NewLeadCluster(geminiClient, embedStore, cfg.GeminiEmbedThreshold)
			}
			if cfg.ParserGeminiEngageMedium && geminiClient != nil {
				warmExtras.EngageMedium = geminiClient
			}
			deps.warmPath = warmpath.NewService(warmpath.Config{
				AnalyzeInterval:         cfg.GeminiLeadAnalyzeInterval,
				BatchSize:               cfg.GeminiLeadBatchSize,
				EngageEnabled:           cfg.ParserGeminiEngage,
				EngageMediumEnabled:     cfg.ParserGeminiEngageMedium,
				EnrichEnabled:           cfg.ParserGeminiEnrichSynth,
				PilotTagEnabled:         cfg.ParserPilotTag,
				GeoBlockCountries:       cfg.GeoBlockCountries,
				GeoClassifyEnabled:      cfg.ParserGeoClassify,
				CRMWebhook:              crmWebhook,
				CRMWebhookAfterAnalysis: deferCRMWebhook,
				RetryMaxAttempts:        cfg.WarmAnalysisRetryMax,
				RetryBaseDelay:          cfg.WarmAnalysisRetryBase,
				PendingRescanInterval:   cfg.WarmAnalysisPendingScanInterval,
				PendingStaleAge:         cfg.WarmAnalysisPendingStale,
				ShutdownDrainTimeout:    cfg.WarmAnalysisShutdownDrain,
			}, warmCapturer, leadPatcher, geminiClient, reg, warmExtras)
			slog.Info("warm path gemini enabled",
				"analyze_interval", cfg.GeminiLeadAnalyzeInterval,
				"batch_size", cfg.GeminiLeadBatchSize,
				"sync_geo", cfg.ParserGeminiSyncGeo,
				"warm_embed_prescan", cfg.ParserWarmEmbedPrescan,
				"warm_embed_cluster", cfg.ParserWarmEmbedCluster,
				"engage_medium", cfg.ParserGeminiEngageMedium,
			)
		}
	}

	if cfg.ParserEntityLinkSuggest && geminiClient != nil {
		if entityStore, ok := entityRecorder.(*sink.EntityStore); ok {
			deps.entityLinkSuggest = warmpath.NewEntityLinkSuggestService(
				warmpath.EntityLinkSuggestConfig{
					Interval: cfg.EntityLinkSuggestInterval,
				},
				entityStore,
				geminiClient,
				entityStore,
			)
			slog.Info("entity link suggest enabled", "interval", cfg.EntityLinkSuggestInterval)
		}
	}

	if cfg.ParserEntityGeminiEnabled && geminiClient != nil && entityRecorder != nil {
		if reader, ok := entityRecorder.(entity.DocReader); ok {
			if patcher, ok := entityRecorder.(entity.ClassificationPatcher); ok {
				entityClassifyCapturer = warmpath.NewEntityClassifyCapturer(cfg.EntityGeminiQueueSize)
				outreachEnabled := cfg.ParserGeminiEngage || cfg.ParserEntityOutreachNarrative
				var classifyExtras warmpath.EntityClassifyExtras
				if force, ok := entityRecorder.(entity.ClassifyForceLister); ok {
					classifyExtras.Force = force
				}
				if lowConf, ok := entityRecorder.(entity.LowConfidenceHotLister); ok {
					classifyExtras.LowConf = lowConf
				}
				if deps.mongoClient != nil {
					if leadMongo, err := sink.NewMongoStoreFromClient(ctx, deps.mongoClient, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots); err == nil {
						classifyExtras.LeadHeat = leadMongo
						classifyExtras.Contacts = leadMongo
						classifyExtras.Outreach = leadMongo
					}
				}
				if outreachEnabled {
					classifyExtras.Narrator = geminiClient
				}
				deps.entityClassify = warmpath.NewEntityClassifyService(
					warmpath.EntityClassifyConfig{
						AnalyzeInterval:          cfg.EntityGeminiInterval,
						Debounce:                 cfg.EntityGeminiDebounce,
						LowConfidenceDebounce:    cfg.EntityGeminiLowConfidenceDebounce,
						OutreachNarrativeEnabled: outreachEnabled,
					},
					entityClassifyCapturer,
					reader,
					patcher,
					geminiClient,
					classifyExtras,
				)
				slog.Info("entity classify gemini enabled",
					"analyze_interval", cfg.EntityGeminiInterval,
					"debounce", cfg.EntityGeminiDebounce,
				)
			}
		}
	}

	if inner != nil {
		storeInner := inner
		if crmWebhook != nil && !deferCRMWebhook {
			storeInner = sink.AttachWebhook(inner, crmWebhook)
			slog.Info("crm webhook enabled", "url", "set", "after_analysis", false)
		} else if deferCRMWebhook {
			// Same WebhookClient instance is passed to warmpath; hot-path Upsert must not notify.
			slog.Info("crm webhook deferred until warm path analysis done")
		}
		deps.bulkStore = sink.NewBulkStore(storeInner, 50, 2*time.Second)
	} else {
		slog.Warn("leads will not be persisted without MONGO_URI or PARSER_EXPORT_JSON")
	}

	if geminiClient != nil && !geminiDefer {
		if cfg.ParserEmbedPrescan {
			embedPrescan = gemini.NewPrescan(geminiClient, cfg.GeminiEmbedPainMin, cfg.GeminiEmbedSpamMin)
		}
		if cfg.ParserEmbedCluster && embedStore != nil {
			leadCluster = gemini.NewLeadCluster(geminiClient, embedStore, cfg.GeminiEmbedThreshold)
		}
	}

	icpEnabled := cfg.ParserICPClassify && geminiClient != nil && !geminiDefer
	// Tgweb sync ICP stays on under gemini defer so site leads are gated before Mongo write.
	icpTgWebEnabled := cfg.ParserICPClassifyTgWeb && geminiClient != nil
	geoEnabled := cfg.ParserGeoClassify && geminiClient != nil && (!geminiDefer || cfg.ParserGeminiSyncGeo)
	engageEnabled := cfg.ParserGeminiEngage && geminiClient != nil && !geminiDefer
	enrichSynthEnabled := cfg.ParserGeminiEnrichSynth && geminiClient != nil && !geminiDefer
	prescanEnabled := cfg.ParserEmbedPrescan && embedPrescan != nil && !geminiDefer
	clusterEnabled := cfg.ParserEmbedCluster && leadCluster != nil && !geminiDefer

	if cfg.ParserTgWebPrescanMode != "" {
		slog.Info("tgweb prescan mode", "mode", scoring.ParseTgWebPrescanMode(cfg.ParserTgWebPrescanMode))
	}
	if icpTgWebEnabled {
		slog.Info("tgweb sync icp enabled", "gemini_defer", geminiDefer)
	}
	leadStore := sink.Store(sink.NewStubStore())
	if deps.bulkStore != nil {
		leadStore = deps.bulkStore
	}
	deps.processor = &pipeline.Processor{
		Registry:              reg,
		Seen:                  dedup.NewSeenCache(50_000, 24*time.Hour),
		Store:                 leadStore,
		MX:                    mx,
		Junk:                  junkCapturer,
		SourceRep:             sourceRep,
		KeywordStats:          keywordStats,
		ICP:                   geminiClient,
		ICPEnabled:            icpEnabled,
		ICPTgWebEnabled:       icpTgWebEnabled,
		TgWebPrescanMode:      scoring.ParseTgWebPrescanMode(cfg.ParserTgWebPrescanMode),
		Geo:                   geminiClient,
		GeoEnabled:            geoEnabled,
		GeoBlockCountries:     cfg.GeoBlockCountries,
		Engage:                geminiClient,
		EngageEnabled:         engageEnabled,
		Prescan:               embedPrescan,
		PrescanEnabled:        prescanEnabled,
		LeadCluster:           leadCluster,
		LeadClusterEnabled:    clusterEnabled,
		Enricher:              enricher,
		EnrichSynth:           geminiClient,
		EnrichSynthEnabled:    enrichSynthEnabled,
		TimeDecayEnabled:      cfg.ParserTimeDecay,
		PilotTagEnabled:       cfg.ParserPilotTag,
		LeadStatusEnabled:     cfg.ParserLeadStatusEnabled,
		GeminiDefer:           geminiDefer,
		WarmPath:              warmCapturer,
		EntityClassify:        entityClassifyCapturer,
		EntityClassifyEnabled: cfg.ParserEntityGeminiEnabled && entityClassifyCapturer != nil,
		EntityRecorder:        entityRecorder,
		EntitySightings:       cfg.ParserEntitySightings && entityRecorder != nil,
		CrossSourceHot:        cfg.ParserCrossSourceHot && cfg.ParserEntitySightings && entityRecorder != nil,
		CrossSourceWindow:     cfg.CrossSourceHotWindow,
		CrossSourceBoost:      cfg.CrossSourceHotBoost,
		EntityHeatEnabled:     cfg.ParserEntityHeatEnabled && cfg.ParserEntitySightings && entityRecorder != nil,
		EntityHeat:            entityHeatFromConfig(cfg),
		HardRejectShadowPct:   cfg.HardRejectShadowPct,
		SeedFeedback: seedfeedback.NewRecorder(seedfeedback.Config{
			Enabled:      cfg.ParserSeedFeedback,
			RegistryPath: cfg.SourceRegistryPath,
			MinHeatTier:  cfg.ParserSeedFeedbackMinHeat,
		}),
		TelegramThread: entity.NewThreadBuffer(entity.DefaultTelegramThreadWindow),
	}

	if cfg.MetricsAddr != "" {
		metrics.StartMetricsServer(ctx, cfg.MetricsAddr)
		slog.Info("prometheus metrics endpoint enabled", "addr", cfg.MetricsAddr)
	}

	return deps, nil
}

func (d *runtimeDeps) flushStore(ctx context.Context) {
	// Flush bulk Mongo buffer only; warm/cold workers drain their own queues on ctx cancel.
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
	if deps.warmPath != nil {
		deps.warmPath.Run(ctx, &wg)
	}
	if deps.entityClassify != nil {
		deps.entityClassify.Run(ctx, &wg)
	}
	if deps.entityLinkSuggest != nil {
		deps.entityLinkSuggest.Run(ctx, &wg)
	}

	pool := pipeline.NewPool(cfg.WorkerCount, deps.processor, cfg.ProcessorTaskTimeout)
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
	if cfg.TelegramDryRun {
		slog.Warn("telegram dry-run: emitting fixtures only; pipeline skips fixture:* (no Mongo writes)")
	}
	pr, pw := io.Pipe()

	sidecarCtx, sidecarCancel := context.WithCancel(ctx)
	defer sidecarCancel()

	errCh := make(chan error, 1)
	go func() {
		err := telethon.Run(sidecarCtx, telethon.Options{
			ConfigPath: cfg.TelegramConfigPath,
			PythonBin:  cfg.TelethonPython,
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
