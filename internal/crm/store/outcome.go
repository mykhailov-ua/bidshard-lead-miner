package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrInvalidOutcome = errors.New("invalid outcome")

const (
	OutcomeContacted         = "contacted"
	OutcomeReplied           = "replied"
	OutcomePilotStarted      = "pilot_started"
	OutcomeMigrationImported = "migration_imported"
)

var allowedOutcomes = map[string]struct{}{
	OutcomeContacted:         {},
	OutcomeReplied:           {},
	OutcomePilotStarted:      {},
	OutcomeMigrationImported: {},
}

// NormalizeOutcome validates downstream CRM outcome labels (pilots, not accepts).
func NormalizeOutcome(raw string) (string, error) {
	outcome := strings.ToLower(strings.TrimSpace(raw))
	if outcome == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidOutcome)
	}
	if _, ok := allowedOutcomes[outcome]; !ok {
		return "", fmt.Errorf("%w: %q", ErrInvalidOutcome, raw)
	}
	return outcome, nil
}

// SetOutcome records operator downstream result on lead + crm meta.
func (s *LeadStore) SetOutcome(ctx context.Context, hashID, outcome, note string) error {
	if s == nil || s.leads == nil {
		return fmt.Errorf("lead store not initialized")
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return fmt.Errorf("hash_id empty")
	}
	resolved, err := s.ResolveHashID(ctx, hashID)
	if err != nil {
		return err
	}
	hashID = resolved
	normalized, err := NormalizeOutcome(outcome)
	if err != nil {
		return err
	}
	note = strings.TrimSpace(note)
	if len(note) > maxNoteLen {
		note = note[:maxNoteLen-3] + "..."
	}
	now := time.Now().UTC()

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	var leadSource string
	var leadRow struct {
		Source string `bson:"source"`
	}
	if err := s.leads.FindOne(writeCtx, bson.M{"hash_id": hashID}).Decode(&leadRow); err == nil {
		leadSource = strings.TrimSpace(leadRow.Source)
	}

	leadSet := bson.M{
		"outcome":    normalized,
		"outcome_at": now,
	}
	if note != "" {
		leadSet["outcome_note"] = note
	}
	res, err := s.leads.UpdateOne(writeCtx, bson.M{"hash_id": hashID}, bson.M{"$set": leadSet})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}

	if s.leadMeta != nil {
		metaSet := bson.M{
			"outcome":    normalized,
			"outcome_at": now,
			"updated_at": now,
		}
		if note != "" {
			metaSet["outcome_note"] = note
		}
		_, err = s.leadMeta.UpdateOne(writeCtx,
			bson.M{"hash_id": hashID},
			bson.M{
				"$set":         metaSet,
				"$setOnInsert": bson.M{"hash_id": hashID},
			},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	if leadSource != "" && s.sourceStats != nil {
		sink.RecordOutcomeOnCollection(s.sourceStats, leadSource, normalized)
	}
	return nil
}

type OutcomeSourceCount struct {
	Source  string `json:"source"`
	Outcome string `json:"outcome"`
	Count   int64  `json:"count"`
}

// AggregateOutcomeBySource groups non-empty lead outcomes by source for dork/source reports.
func (s *LeadStore) AggregateOutcomeBySource(ctx context.Context) ([]OutcomeSourceCount, error) {
	if s == nil || s.leads == nil {
		return nil, fmt.Errorf("lead store not initialized")
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"outcome": bson.M{"$exists": true, "$ne": ""}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   bson.M{"source": "$source", "outcome": "$outcome"},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}
	cur, err := s.leads.Aggregate(queryCtx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var rows []struct {
		ID struct {
			Source  string `bson:"source"`
			Outcome string `bson:"outcome"`
		} `bson:"_id"`
		Count int64 `bson:"count"`
	}
	if err := cur.All(queryCtx, &rows); err != nil {
		return nil, err
	}
	out := make([]OutcomeSourceCount, 0, len(rows))
	for _, row := range rows {
		if row.ID.Source == "" || row.ID.Outcome == "" || row.Count == 0 {
			continue
		}
		out = append(out, OutcomeSourceCount{
			Source:  row.ID.Source,
			Outcome: row.ID.Outcome,
			Count:   row.Count,
		})
	}
	return out, nil
}
