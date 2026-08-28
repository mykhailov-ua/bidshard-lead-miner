package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/mongo"
)

// DiscoverFeedbackResult summarizes one feedback loop pass.
type DiscoverFeedbackResult struct {
	OutcomeReportPath  string `json:"outcome_report_path,omitempty"`
	KeywordTunePath    string `json:"keyword_tune_path,omitempty"`
	KeywordTuneApplied string `json:"keyword_tune_applied,omitempty"`
	SalesFeedbackPath  string `json:"sales_feedback_ru_path,omitempty"`
	PrunedDorks        int    `json:"pruned_dorks"`
}

type sourceStatsReader interface {
	ListAll(ctx context.Context) ([]sink.SourceStatsDoc, error)
}

type keywordStatsReader interface {
	ListAll(ctx context.Context, limit int) ([]sink.KeywordStatDoc, error)
}

// RunFeedbackCommand loads runtime deps and runs one discover feedback pass.
func RunFeedbackCommand(ctx context.Context, cfg config.Config) (DiscoverFeedbackResult, error) {
	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return DiscoverFeedbackResult{}, err
	}
	defer deps.closeMongo(ctx)
	return RunDiscoverFeedback(ctx, cfg, deps.mongoClient, deps.sourceStats, deps.keywordStats, deps.registry)
}

// RunDiscoverFeedback writes outcome/keyword reports and disables underperforming dorks.
func RunDiscoverFeedback(ctx context.Context, cfg config.Config, client *mongo.Client, stats sourceStatsReader, keywordStats keywordStatsReader, reg *scoring.Registry) (DiscoverFeedbackResult, error) {
	var result DiscoverFeedbackResult
	if stats == nil {
		return result, nil
	}

	var sourceDocs []sink.SourceStatsDoc
	if docs, err := stats.ListAll(ctx); err == nil {
		sourceDocs = docs
	}

	var outcomeRows []discover.OutcomeSourceRow
	if client != nil && cfg.MongoCollection != "" {
		leads := client.Database(cfg.MongoDB).Collection(cfg.MongoCollection)
		if counts, err := sink.AggregateOutcomeBySource(ctx, leads, cfg.ShutdownTimeout); err == nil {
			for _, c := range counts {
				outcomeRows = append(outcomeRows, discover.OutcomeSourceRow{
					Source:  c.Source,
					Outcome: c.Outcome,
					Count:   c.Count,
				})
			}
		}
	}

	channelsPath := cfg.TelegramChannelsPath
	if channelsPath == "" {
		channelsPath = "data/runtime/discovered_telegram_channels.json"
	}
	outDir := cfg.GeminiKeywordDiffDir
	if outDir == "" {
		outDir = "data/suggestions"
	}

	reportPath, err := discover.WriteOutcomeDorkReport(channelsPath, outDir, sourceDocs, outcomeRows)
	if err != nil {
		return result, err
	}
	result.OutcomeReportPath = reportPath

	dorkRows, err := discover.BuildOutcomeDorkRows(channelsPath, sourceDocs, outcomeRows)
	if err != nil {
		return result, err
	}
	pruneCfg := discover.DorkPruneConfig{
		MinRaw:              cfg.DorkDisableMinRaw,
		MaxAcceptRate:       cfg.DorkDisableMaxAcceptRate,
		RequireZeroOutcomes: true,
	}
	if pruneCfg.MinRaw <= 0 {
		pruneCfg = discover.DefaultDorkPruneConfig()
	}
	if cfg.DorkDisableMaxAcceptRate > 0 {
		pruneCfg.MaxAcceptRate = cfg.DorkDisableMaxAcceptRate
	}
	toDisable := discover.EvaluateDorkPrune(dorkRows, pruneCfg)
	if len(toDisable) > 0 {
		path := cfg.DisabledDorksPath
		if path == "" {
			path = dorkdisable.DefaultPath
		}
		audit := make([]string, 0, len(toDisable))
		for _, dork := range toDisable {
			audit = append(audit, fmt.Sprintf("%s: accept_rate<=%.2f raw>=%d outcomes=0", dork, pruneCfg.MaxAcceptRate, pruneCfg.MinRaw))
		}
		if err := dorkdisable.Save(path, toDisable, audit); err != nil {
			return result, err
		}
		result.PrunedDorks = len(toDisable)
		slog.Info("discover feedback pruned dorks", "count", len(toDisable), "path", path)
	}

	var tuneRows []discover.KeywordTuneRow
	if keywordStats != nil {
		docs, err := keywordStats.ListAll(ctx, 500)
		if err != nil {
			return result, err
		}
		weights := keywordWeightsFromRegistry(reg)
		tuneRows = discover.BuildKeywordTuneRows(docs, weights)
		if len(tuneRows) > 0 {
			tunePath, err := discover.WriteKeywordTuneReport(outDir, tuneRows)
			if err != nil {
				return result, err
			}
			result.KeywordTunePath = tunePath

			if cfg.ParserKeywordTuneAutoApply {
				if summary, err := applyKeywordTuneAuto(ctx, cfg, tuneRows, reg); err != nil {
					slog.Warn("keyword tune auto-apply failed", "error", err)
				} else if summary != "" {
					result.KeywordTuneApplied = summary
					slog.Info("keyword tune auto-applied", "summary", summary)
				}
			}
		}
	}

	if cfg.ParserSalesExportRU {
		if path, err := writeSalesFeedbackRU(ctx, cfg, dorkRows, tuneRows); err != nil {
			slog.Warn("sales feedback RU export failed", "error", err)
		} else {
			result.SalesFeedbackPath = path
		}
	}

	return result, nil
}

func keywordWeightsFromRegistry(reg *scoring.Registry) map[string]int {
	if reg == nil {
		return map[string]int{}
	}
	return reg.KeywordWeights()
}
