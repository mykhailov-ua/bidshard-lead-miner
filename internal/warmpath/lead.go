package warmpath

import (
	"time"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sink"
)

// leadForCRM builds webhook payload from the warm-path queue event plus analysis patch.
// We do not re-read Mongo: contacts/snippet come from accept-time Event (same as hot-path upsert).
func leadForCRM(ev Event, patch sink.LeadAnalysisPatch) model.Lead {
	status := patch.Status
	if status == "" {
		status = "new"
	}
	ts := ev.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return model.Lead{
		TS:              ts.UTC(),
		RoundID:         ev.RoundID,
		HashID:          ev.HashID,
		Priority:        patch.Priority,
		Score:           patch.Score,
		Source:          ev.Source,
		Title:           ev.Title,
		Contacts:        append([]string(nil), ev.Contacts...),
		Matched:         append([]string(nil), ev.Matched...),
		Snippet:         ev.Snippet,
		ICP:             patch.ICP,
		Hot:             patch.Hot,
		SpendTier:       patch.SpendTier,
		ICPWhy:          patch.ICPWhy,
		GeoCountry:      patch.GeoCountry,
		CompanyCountry:  patch.CompanyCountry,
		CompanyName:     patch.CompanyName,
		GeoSignals:      append([]string(nil), patch.GeoSignals...),
		GeoWhy:          patch.GeoWhy,
		WhoisCountry:    ev.RDAPCountry,
		DomainAgeDays:   ev.DomainAgeDays,
		DisplayName:     ev.DisplayName,
		Stack:           append([]string(nil), ev.Stack...),
		Status:          status,
		AnalysisStatus:  patch.AnalysisStatus,
		PilotQualified:  patch.PilotQualified,
		PilotWhy:        patch.PilotWhy,
		Tags:            append([]string(nil), patch.Tags...),
		OutreachChannel: patch.OutreachChannel,
		OutreachSubject: patch.OutreachSubject,
		OutreachAngle:   patch.OutreachAngle,
		OutreachDraft:   patch.OutreachDraft,
		CompanyType:     patch.CompanyType,
		EnrichSummary:   patch.EnrichSummary,
		GeoConfidence:   patch.GeoConfidence,
		EntityID:        ev.EntityID,
		EntityHeat:      ev.EntityHeat,
		HeatTier:        ev.HeatTier,
		ContactQuality:  patch.ContactQuality,
		ContactChannel:  patch.ContactChannel,
		NextAction:      patch.NextAction,
	}
}
