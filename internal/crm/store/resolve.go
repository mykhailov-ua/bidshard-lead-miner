package store

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrAmbiguousHash = errors.New("ambiguous hash prefix")

var hashIDPattern = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// ResolveHashID accepts a full hash_id or unique hex prefix from /new list output.
func (s *LeadStore) ResolveHashID(ctx context.Context, raw string) (string, error) {
	if s == nil || s.leads == nil {
		return "", fmt.Errorf("lead store not initialized")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("hash_id empty")
	}
	if !hashIDPattern.MatchString(raw) {
		return "", fmt.Errorf("hash_id must be hex")
	}

	if _, err := s.GetByHashID(ctx, raw); err == nil {
		return raw, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	cur, err := s.leads.Find(queryCtx,
		bson.M{"hash_id": bson.M{"$regex": "^" + raw}},
		options.Find().SetLimit(2).SetProjection(bson.M{"hash_id": 1}),
	)
	if err != nil {
		return "", err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var matches []string
	for cur.Next(queryCtx) {
		var doc struct {
			HashID string `bson:"hash_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return "", err
		}
		matches = append(matches, doc.HashID)
	}
	if err := cur.Err(); err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return "", ErrAmbiguousHash
	}
}
