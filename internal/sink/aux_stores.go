package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SourceStatsDoc struct {
	Source             string    `bson:"source"`
	Accepted           int       `bson:"accepted"`
	Junk               int       `bson:"junk"`
	OutcomeContacted   int       `bson:"outcome_contacted,omitempty"`
	OutcomeReplied     int       `bson:"outcome_replied,omitempty"`
	OutcomePilot       int       `bson:"outcome_pilot,omitempty"`
	OutcomeMigration   int       `bson:"outcome_migration,omitempty"`
	Updated            time.Time `bson:"updated"`
}

type SourceStatsStore struct {
	coll *mongo.Collection
}

func ConnectSourceStats(ctx context.Context, client *mongo.Client, dbName, collection string) (*SourceStatsStore, error) {
	coll, err := connectIndexedCollectionOne(ctx, client, dbName, collection, "source_stats", mongo.IndexModel{
		Keys: bson.D{{Key: "source", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return &SourceStatsStore{coll: coll}, nil
}

func (s *SourceStatsStore) RecordAccepted(source string) {
	s.bump(source, true)
}

func (s *SourceStatsStore) RecordJunk(source string) {
	s.bump(source, false)
}

func (s *SourceStatsStore) bump(source string, accepted bool) {
	if s == nil || source == "" {
		return
	}
	field := "junk"
	if accepted {
		field = "accepted"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = s.coll.UpdateOne(ctx,
		bson.M{"source": source},
		bson.M{
			"$inc": bson.M{field: 1},
			"$set": bson.M{"updated": time.Now().UTC()},
		},
		options.Update().SetUpsert(true),
	)
}

func (s *SourceStatsStore) ListAll(ctx context.Context) ([]SourceStatsDoc, error) {
	if s == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cur, err := s.coll.Find(ctx, bson.M{}, options.Find().SetLimit(500))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []SourceStatsDoc
	return docs, cur.All(ctx, &docs)
}

func (s *SourceStatsStore) RecordOutcome(source, outcome string) {
	RecordOutcomeOnCollection(s.coll, source, outcome)
}

func RecordOutcomeOnCollection(coll *mongo.Collection, source, outcome string) {
	if coll == nil || source == "" {
		return
	}
	field := outcomeField(outcome)
	if field == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = coll.UpdateOne(ctx,
		bson.M{"source": source},
		bson.M{
			"$inc": bson.M{field: 1},
			"$set": bson.M{"updated": time.Now().UTC()},
		},
		options.Update().SetUpsert(true),
	)
}

func outcomeField(outcome string) string {
	switch outcome {
	case "contacted":
		return "outcome_contacted"
	case "replied":
		return "outcome_replied"
	case "pilot_started":
		return "outcome_pilot"
	case "migration_imported":
		return "outcome_migration"
	default:
		return ""
	}
}

func SourceStatsBoost(doc SourceStatsDoc) int {
	total := doc.Accepted + doc.Junk
	boost := 0
	if total >= 20 {
		ratio := float64(doc.Accepted) / float64(total)
		switch {
		case ratio >= 0.35:
			boost = 8
		case ratio >= 0.2:
			boost = 4
		case ratio < 0.08:
			boost = -4
		}
	}
	boost += outcomeBoost(doc)
	return boost
}

func outcomeBoost(doc SourceStatsDoc) int {
	boost := 0
	if doc.OutcomeMigration > 0 {
		boost += 10
	}
	if doc.OutcomePilot > 0 {
		boost += 8
	}
	if doc.OutcomeReplied > 0 {
		boost += 4
	}
	if doc.OutcomeContacted > 0 {
		boost += 2
	}
	if boost > 12 {
		return 12
	}
	return boost
}

func (s *SourceStatsStore) Boost(source string) int {
	if s == nil {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var doc SourceStatsDoc
	err := s.coll.FindOne(ctx, bson.M{"source": source}).Decode(&doc)
	if err != nil {
		return 0
	}
	return SourceStatsBoost(doc)
}

type CrmBoostDoc struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TS          time.Time          `bson:"ts" json:"ts"`
	JunkID      string             `bson:"junk_id,omitempty" json:"junk_id,omitempty"`
	Source      string             `bson:"source" json:"source"`
	Snippet     string             `bson:"snippet" json:"snippet"`
	ContactHint string             `bson:"contact_hint,omitempty" json:"contact_hint,omitempty"`
	Why         string             `bson:"why" json:"why"`
	Priority    string             `bson:"priority" json:"priority"` // High
	Status      string             `bson:"status" json:"status"`     // pending|promoted|dismissed|merged
	LeadHashID  string             `bson:"lead_hash_id,omitempty" json:"lead_hash_id,omitempty"`
	OutcomeWhy  string             `bson:"outcome_why,omitempty" json:"outcome_why,omitempty"`
	ResolvedAt  time.Time          `bson:"resolved_at,omitempty" json:"resolved_at,omitempty"`
}

type CrmBoostStore struct {
	coll *mongo.Collection
}

func ConnectCrmBoost(ctx context.Context, client *mongo.Client, dbName, collection string) (*CrmBoostStore, error) {
	coll, err := connectIndexedCollectionOne(ctx, client, dbName, collection, "crm_boosts", mongo.IndexModel{
		Keys: bson.D{{Key: "ts", Value: -1}},
	})
	if err != nil {
		return nil, err
	}
	return &CrmBoostStore{coll: coll}, nil
}

func (s *CrmBoostStore) Insert(ctx context.Context, doc CrmBoostDoc) error {
	if s == nil {
		return nil
	}
	if doc.TS.IsZero() {
		doc.TS = time.Now().UTC()
	}
	if doc.Status == "" {
		doc.Status = CrmBoostPending
	}
	if doc.Priority == "" {
		doc.Priority = "High"
	}
	_, err := s.coll.InsertOne(ctx, doc)
	return err
}

type EmbeddingDoc struct {
	Key         string    `bson:"key"`
	Vector      []float32 `bson:"vector"`
	TS          time.Time `bson:"ts"`
	Kind        string    `bson:"kind,omitempty"`
	HashID      string    `bson:"hash_id,omitempty"`
	SemanticDup bool      `bson:"semantic_dup,omitempty"`
	DupOf       string    `bson:"dup_of,omitempty"`
}

const (
	EmbedKindJunk = "junk"
	EmbedKindLead = "lead"
)

func LeadEmbedKey(hashID string) string {
	return "lead:" + hashID
}

type EmbeddingStore struct {
	coll *mongo.Collection
}

func ConnectEmbeddingStore(ctx context.Context, client *mongo.Client, dbName, collection string) (*EmbeddingStore, error) {
	coll, err := connectIndexedCollectionOne(ctx, client, dbName, collection, "snippet_embeddings", mongo.IndexModel{
		Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}
	return &EmbeddingStore{coll: coll}, nil
}

func (s *EmbeddingStore) Upsert(ctx context.Context, doc EmbeddingDoc) error {
	if s == nil {
		return nil
	}
	if doc.TS.IsZero() {
		doc.TS = time.Now().UTC()
	}
	_, err := s.coll.UpdateOne(ctx,
		bson.M{"key": doc.Key},
		bson.M{"$set": doc},
		options.Update().SetUpsert(true),
	)
	return err
}

func (s *EmbeddingStore) RecentVectors(ctx context.Context, limit int) ([]EmbeddingDoc, error) {
	return s.RecentVectorsByKind(ctx, "", limit)
}

func (s *EmbeddingStore) RecentVectorsByKind(ctx context.Context, kind string, limit int) ([]EmbeddingDoc, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	filter := bson.M{}
	if kind != "" {
		filter["kind"] = kind
	}
	cur, err := s.coll.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "ts", Value: -1}}).
		SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(ctx) }()
	var out []EmbeddingDoc
	return out, cur.All(ctx, &out)
}
