package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/salesexport"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/suggestions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SalesExportResult lists written Russian sales JSON paths.
type SalesExportResult struct {
	LeadsPath    string `json:"leads_path,omitempty"`
	JunkPath     string `json:"junk_path,omitempty"`
	FeedbackPath string `json:"feedback_path,omitempty"`
}

// RunSalesExport writes Russian JSON bundles for sales managers.
func RunSalesExport(ctx context.Context, cfg config.Config, limit int, minScore int) (SalesExportResult, error) {
	var result SalesExportResult
	if limit <= 0 {
		limit = 50
	}
	outDir := cfg.SalesExportDir
	if outDir == "" {
		outDir = "data/export/sales"
	}

	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return result, err
	}
	defer deps.closeMongo(ctx)

	var geminiClient *gemini.Client
	if cfg.GeminiAPIKey != "" {
		geminiClient, _ = gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel, gemini.ClientOptionsFrom(cfg)...)
	}

	if deps.mongoClient != nil && cfg.MongoCollection != "" {
		leadsColl := deps.mongoClient.Database(cfg.MongoDB).Collection(cfg.MongoCollection)
		docs, err := listSalesLeads(ctx, leadsColl, limit, minScore)
		if err != nil {
			return result, err
		}
		cards := make([]salesexport.LeadCardRU, 0, len(docs))
		for _, doc := range docs {
			cards = append(cards, salesexport.LeadCardFromDoc(doc))
		}
		if geminiClient != nil && len(cards) > 0 {
			if localized, err := geminiClient.LocalizeLeadCardsRU(ctx, cards); err == nil {
				cards = localized
			} else {
				slog.Warn("sales leads RU localize failed", "error", err)
			}
		}
		bundle := salesexport.LeadsBundleRU{
			Title:       "Карточки лидов для отдела продаж",
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			LeadCount:   len(cards),
			Leads:       cards,
		}
		if geminiClient == nil {
			bundle.Note = "Gemini не настроен: тексты outreach могут быть на английском"
		}
		path, err := salesexport.WriteJSON(outDir, "leads_ru", bundle)
		if err != nil {
			return result, err
		}
		result.LeadsPath = path
	}

	if deps.mongoClient != nil && cfg.ColdJunkCollection != "" {
		junkStore, err := sink.ConnectJunkStore(ctx, deps.mongoClient, cfg.MongoDB, cfg.ColdJunkCollection, cfg.ColdReportCollection)
		if err == nil && junkStore != nil {
			doc, err := junkStore.LatestJunkReport(ctx)
			if err == nil && doc.SampleCount > 0 {
				report := salesexport.JunkReportFromDoc(doc)
				if geminiClient != nil {
					if localized, err := geminiClient.LocalizeJunkReportRU(ctx, doc); err == nil {
						report = localized
					} else {
						slog.Warn("sales junk report RU localize failed", "error", err)
					}
				}
				path, err := salesexport.WriteJSON(outDir, "junk_report_ru", report)
				if err != nil {
					return result, err
				}
				result.JunkPath = path
			}
		}
	}

	return result, nil
}

func listSalesLeads(ctx context.Context, coll *mongo.Collection, limit, minScore int) ([]sink.LeadDoc, error) {
	if coll == nil {
		return nil, nil
	}
	filter := bson.M{}
	if minScore > 0 {
		filter["score"] = bson.M{"$gte": minScore}
	}
	cur, err := coll.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "score", Value: -1}, {Key: "ts", Value: -1}}).
		SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []sink.LeadDoc
	return docs, cur.All(ctx, &docs)
}

func writeSalesFeedbackRU(ctx context.Context, cfg config.Config, dorkRows []discover.OutcomeDorkRow, tuneRows []discover.KeywordTuneRow) (string, error) {
	outDir := cfg.SalesExportDir
	if outDir == "" {
		outDir = "data/export/sales"
	}
	bundle := salesexport.FeedbackBundleFromDiscover(dorkRows, tuneRows, time.Now().UTC().Format(time.RFC3339))
	return salesexport.WriteJSON(outDir, "feedback_ru", bundle)
}

func reloadKeywordRegistry(ctx context.Context, cfg config.Config, reg *scoring.Registry) {
	if reg == nil {
		return
	}
	overlays := []string{cfg.KeywordsGrayPath}
	overlays = append(overlays, config.KeywordOverlayPaths(cfg.KeywordsLocale, cfg.KeywordsLocalePath)...)
	if err := reg.LoadWithOverlays(ctx, cfg.KeywordsJSONPath, overlays...); err != nil {
		slog.Warn("keyword registry reload failed", "error", err)
		return
	}
	slog.Info("keyword registry reloaded after tune auto-apply")
}

func applyKeywordTuneAuto(ctx context.Context, cfg config.Config, tuneRows []discover.KeywordTuneRow, reg *scoring.Registry) (string, error) {
	if len(tuneRows) == 0 || cfg.KeywordsJSONPath == "" {
		return "", nil
	}
	summary, err := suggestions.ApplyKeywordTuneAuto(cfg.KeywordsJSONPath, tuneRows, suggestions.KeywordTuneAutoApplyOptions{
		MaxPerWeek: cfg.KeywordTuneAutoApplyMaxWeek,
	}, false)
	if err != nil {
		return "", err
	}
	reloadKeywordRegistry(ctx, cfg, reg)
	return summary, nil
}
