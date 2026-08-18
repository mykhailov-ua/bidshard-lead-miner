package sink

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type JunkAnalysis struct {
	AnalyzedAt  time.Time `bson:"analyzed_at" json:"analyzed_at"`
	Category    string    `bson:"category" json:"category"`
	Why         string    `bson:"why" json:"why"`
	Suggestions []string  `bson:"suggestions,omitempty" json:"suggestions,omitempty"`
}

type JunkDoc struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TS           time.Time          `bson:"ts" json:"ts"`
	RoundID      string             `bson:"round_id" json:"round_id"`
	Source       string             `bson:"source" json:"source"`
	Title        string             `bson:"title,omitempty" json:"title,omitempty"`
	Snippet      string             `bson:"snippet" json:"snippet"`
	ContactHint  string             `bson:"contact_hint,omitempty" json:"contact_hint,omitempty"`
	Reason       string             `bson:"reason" json:"reason"`
	ReasonDetail string             `bson:"reason_detail,omitempty" json:"reason_detail,omitempty"`
	Score        int                `bson:"score,omitempty" json:"score,omitempty"`
	Matched      []string           `bson:"matched,omitempty" json:"matched,omitempty"`
	Analysis     *JunkAnalysis      `bson:"analysis,omitempty" json:"analysis,omitempty"`
}

type JunkReportDoc struct {
	ID                      primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	TS                      time.Time          `bson:"ts" json:"ts"`
	PeriodFrom              time.Time          `bson:"period_from" json:"period_from"`
	PeriodTo                time.Time          `bson:"period_to" json:"period_to"`
	SampleCount             int                `bson:"sample_count" json:"sample_count"`
	Summary                 string             `bson:"summary" json:"summary"`
	TopReasons              []ReasonCount      `bson:"top_reasons" json:"top_reasons"`
	FalseNegativeCandidates int                `bson:"false_negative_candidates" json:"false_negative_candidates"`
	Recommendations         []string           `bson:"recommendations" json:"recommendations"`
	KeywordSuggestions      []string           `bson:"keyword_suggestions,omitempty" json:"keyword_suggestions,omitempty"`
}

type ReasonCount struct {
	Reason string `bson:"reason" json:"reason"`
	Count  int    `bson:"count" json:"count"`
	Why    string `bson:"why" json:"why"`
}

type JunkStore struct {
	leads   *mongo.Collection
	reports *mongo.Collection
}

func ConnectJunkStore(ctx context.Context, client *mongo.Client, dbName, leadsColl, reportsColl string) (*JunkStore, error) {
	if client == nil {
		return nil, errors.New("mongo client required")
	}
	if dbName == "" {
		dbName = "parser"
	}
	if leadsColl == "" {
		leadsColl = "junk_leads"
	}
	if reportsColl == "" {
		reportsColl = "junk_reports"
	}

	store := &JunkStore{
		leads:   client.Database(dbName).Collection(leadsColl),
		reports: client.Database(dbName).Collection(reportsColl),
	}
	if err := store.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *JunkStore) ensureIndexes(ctx context.Context) error {
	_, err := s.leads.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "ts", Value: -1}}},
		{Keys: bson.D{{Key: "reason", Value: 1}}},
		{Keys: bson.D{{Key: "analysis.analyzed_at", Value: 1}}, Options: options.Index().SetSparse(true)},
	})
	if err != nil {
		return err
	}
	_, err = s.reports.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "ts", Value: -1}},
	})
	return err
}

func (s *JunkStore) InsertMany(ctx context.Context, docs []JunkDoc) error {
	if len(docs) == 0 {
		return nil
	}
	items := make([]any, len(docs))
	for i := range docs {
		items[i] = docs[i]
	}
	_, err := s.leads.InsertMany(ctx, items, options.InsertMany().SetOrdered(false))
	return err
}

func (s *JunkStore) FindPendingAnalysis(ctx context.Context, limit int) ([]JunkDoc, error) {
	if limit <= 0 {
		limit = 20
	}
	cur, err := s.leads.Find(ctx, bson.M{"analysis": bson.M{"$exists": false}},
		options.Find().SetSort(bson.D{{Key: "ts", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []JunkDoc
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *JunkStore) SaveAnalysis(ctx context.Context, id primitive.ObjectID, analysis JunkAnalysis) error {
	_, err := s.leads.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"analysis": analysis}},
	)
	return err
}

func (s *JunkStore) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return s.leads.CountDocuments(ctx, bson.M{"ts": bson.M{"$gte": since}})
}

func (s *JunkStore) ReasonBreakdown(ctx context.Context, since time.Time) ([]ReasonCount, error) {
	pipe := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"ts": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$reason",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		{{Key: "$limit", Value: 12}},
	}
	cur, err := s.leads.Aggregate(ctx, pipe)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	type row struct {
		ID    string `bson:"_id"`
		Count int    `bson:"count"`
	}
	var rows []row
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make([]ReasonCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, ReasonCount{Reason: r.ID, Count: r.Count})
	}
	return out, nil
}

func (s *JunkStore) SampleAnalyzed(ctx context.Context, since time.Time, limit int) ([]JunkDoc, error) {
	if limit <= 0 {
		limit = 30
	}
	cur, err := s.leads.Find(ctx, bson.M{
		"ts":       bson.M{"$gte": since},
		"analysis": bson.M{"$exists": true},
	}, options.Find().
		SetSort(bson.D{{Key: "analysis.analyzed_at", Value: -1}}).
		SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []JunkDoc
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *JunkStore) InsertReport(ctx context.Context, doc JunkReportDoc) error {
	_, err := s.reports.InsertOne(ctx, doc)
	return err
}

func (s *JunkStore) FindByCategorySince(ctx context.Context, category string, since time.Time, limit int) ([]JunkDoc, error) {
	if limit <= 0 {
		limit = 20
	}
	cur, err := s.leads.Find(ctx, bson.M{
		"analysis.category": category,
		"ts":                bson.M{"$gte": since},
	}, options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []JunkDoc
	return out, cur.All(ctx, &out)
}

func (s *JunkStore) MarkSemanticDup(ctx context.Context, id primitive.ObjectID, dupOf string) error {
	_, err := s.leads.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"semantic_dup": true, "semantic_dup_of": dupOf}},
	)
	return err
}
