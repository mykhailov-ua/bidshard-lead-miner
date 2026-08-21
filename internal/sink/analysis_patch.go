package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// LeadAnalysisPatch updates Gemini-derived fields on an existing lead document.
type LeadAnalysisPatch struct {
	HashID          string
	AnalysisStatus  string
	Status          string
	Priority        string
	Score           int
	ICP             string
	Hot             bool
	SpendTier       string
	ICPWhy          string
	GeoCountry      string
	CompanyCountry  string
	CompanyName     string
	GeoSignals      []string
	GeoWhy          string
	Tags            []string
	OutreachChannel string
	OutreachAngle   string
	OutreachDraft   string
	PilotQualified  bool
	PilotWhy        string
	CompanyType     string
	EnrichSummary   string
	GeoConfidence   string
}

// LeadAnalysisPatcher patches deferred Gemini analysis results.
type LeadAnalysisPatcher interface {
	PatchLeadAnalysis(ctx context.Context, patch LeadAnalysisPatch) error
}

func ConnectLeadAnalysisPatcher(ctx context.Context, client *mongo.Client, dbName, collection string, writeSlots int) (LeadAnalysisPatcher, error) {
	return connectMongoStore(ctx, client, dbName, collection, writeSlots, true)
}

func (s *MongoStore) PatchLeadAnalysis(ctx context.Context, patch LeadAnalysisPatch) error {
	if s == nil || patch.HashID == "" {
		return nil
	}

	set := bson.M{
		"analysis_status": patch.AnalysisStatus,
		"analysis_at":     time.Now().UTC(),
	}
	if patch.Status != "" {
		set["status"] = patch.Status
		set["status_at"] = time.Now().UTC()
	}
	if patch.Priority != "" {
		set["priority"] = patch.Priority
	}
	if patch.Score > 0 {
		set["score"] = patch.Score
	}
	if patch.ICP != "" {
		set["icp"] = patch.ICP
	}
	if patch.Hot {
		set["hot"] = patch.Hot
	}
	if patch.SpendTier != "" {
		set["spend_tier"] = patch.SpendTier
	}
	if patch.ICPWhy != "" {
		set["icp_why"] = patch.ICPWhy
	}
	if patch.GeoCountry != "" {
		set["geo_country"] = patch.GeoCountry
	}
	if patch.CompanyCountry != "" {
		set["company_country"] = patch.CompanyCountry
	}
	if patch.CompanyName != "" {
		set["company_name"] = patch.CompanyName
	}
	if len(patch.GeoSignals) > 0 {
		set["geo_signals"] = patch.GeoSignals
	}
	if patch.GeoWhy != "" {
		set["geo_why"] = patch.GeoWhy
	}
	if len(patch.Tags) > 0 {
		set["tags"] = patch.Tags
	}
	if patch.OutreachChannel != "" {
		set["outreach_channel"] = patch.OutreachChannel
	}
	if patch.OutreachAngle != "" {
		set["outreach_angle"] = patch.OutreachAngle
	}
	if patch.OutreachDraft != "" {
		set["outreach_draft"] = patch.OutreachDraft
	}
	if patch.PilotQualified {
		set["pilot_qualified"] = patch.PilotQualified
	}
	if patch.PilotWhy != "" {
		set["pilot_why"] = patch.PilotWhy
	}
	if patch.CompanyType != "" {
		set["company_type"] = patch.CompanyType
	}
	if patch.EnrichSummary != "" {
		set["enrich_summary"] = patch.EnrichSummary
	}
	if patch.GeoConfidence != "" {
		set["geo_confidence"] = patch.GeoConfidence
	}

	if err := s.writeSlots.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.writeSlots.Release(1)

	_, err := s.coll.UpdateOne(ctx,
		bson.M{"hash_id": patch.HashID},
		bson.M{"$set": set},
	)
	return err
}
