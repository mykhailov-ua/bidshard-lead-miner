package sink

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// LatestJunkReport returns the most recent cold-path Gemini junk report.
func (s *JunkStore) LatestJunkReport(ctx context.Context) (JunkReportDoc, error) {
	var doc JunkReportDoc
	if s == nil || s.reports == nil {
		return doc, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := s.reports.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "ts", Value: -1}})).Decode(&doc)
	return doc, err
}
