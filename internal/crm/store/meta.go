package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	maxNoteLen      = 500
	maxTagLen       = 32
	maxNotesList    = 5
	maxNotesPreview = 200
)

type NoteDoc struct {
	HashID       string    `bson:"hash_id"`
	AuthorChatID int64     `bson:"author_chat_id"`
	Text         string    `bson:"text"`
	TS           time.Time `bson:"ts"`
}

type LeadMetaDoc struct {
	HashID           string    `bson:"hash_id"`
	Tags             []string  `bson:"tags,omitempty"`
	ExplainSummary   string    `bson:"explain_summary,omitempty"`
	ExplainCachedAt  time.Time `bson:"explain_cached_at,omitempty"`
	ExplainExpiresAt time.Time `bson:"explain_expires_at,omitempty"`
	UpdatedAt        time.Time `bson:"updated_at"`
}

func normalizeNoteText(text string) (string, error) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "", fmt.Errorf("note text empty")
	}
	if len(text) > maxNoteLen {
		text = text[:maxNoteLen-3] + "..."
	}
	return text, nil
}

func normalizeTag(raw string) (string, error) {
	tag := strings.ToLower(strings.TrimSpace(raw))
	if tag == "" {
		return "", fmt.Errorf("tag empty")
	}
	if len(tag) > maxTagLen {
		return "", fmt.Errorf("tag too long (max %d)", maxTagLen)
	}
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", fmt.Errorf("tag must be alphanumeric, _, or -")
	}
	return tag, nil
}

func (s *LeadStore) AddNote(ctx context.Context, hashID, text string, authorChatID int64) error {
	if s == nil || s.leadNotes == nil {
		return fmt.Errorf("lead notes collection not configured")
	}
	hashID, err := s.resolveExistingHash(ctx, hashID)
	if err != nil {
		return err
	}
	text, err = normalizeNoteText(text)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	_, err = s.leadNotes.InsertOne(writeCtx, NoteDoc{
		HashID:       hashID,
		AuthorChatID: authorChatID,
		Text:         text,
		TS:           time.Now().UTC(),
	})
	return err
}

func (s *LeadStore) ListNotes(ctx context.Context, hashID string, limit int64) ([]NoteDoc, error) {
	if s == nil || s.leadNotes == nil {
		return nil, fmt.Errorf("lead notes collection not configured")
	}
	hashID, err := s.resolveExistingHash(ctx, hashID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > maxNotesList {
		limit = maxNotesList
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	cur, err := s.leadNotes.Find(queryCtx,
		bson.M{"hash_id": hashID},
		options.Find().SetSort(bson.D{{Key: "ts", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	var docs []NoteDoc
	if err := cur.All(queryCtx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *LeadStore) AddTag(ctx context.Context, hashID, tag string) (LeadMetaDoc, error) {
	if s == nil || s.leadMeta == nil {
		return LeadMetaDoc{}, fmt.Errorf("lead crm meta collection not configured")
	}
	hashID, err := s.resolveExistingHash(ctx, hashID)
	if err != nil {
		return LeadMetaDoc{}, err
	}
	tag, err = normalizeTag(tag)
	if err != nil {
		return LeadMetaDoc{}, err
	}

	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	_, err = s.leadMeta.UpdateOne(writeCtx,
		bson.M{"hash_id": hashID},
		bson.M{
			"$addToSet": bson.M{"tags": tag},
			"$set":      bson.M{"updated_at": time.Now().UTC()},
			"$setOnInsert": bson.M{
				"hash_id": hashID,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return LeadMetaDoc{}, err
	}
	return s.GetMeta(ctx, hashID)
}

func (s *LeadStore) GetMeta(ctx context.Context, hashID string) (LeadMetaDoc, error) {
	if s == nil || s.leadMeta == nil {
		return LeadMetaDoc{}, fmt.Errorf("lead crm meta collection not configured")
	}
	hashID, err := s.resolveExistingHash(ctx, hashID)
	if err != nil {
		return LeadMetaDoc{}, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	var doc LeadMetaDoc
	err = s.leadMeta.FindOne(queryCtx, bson.M{"hash_id": hashID}).Decode(&doc)
	if err != nil {
		return LeadMetaDoc{HashID: hashID}, nil
	}
	return doc, nil
}

func (s *LeadStore) resolveExistingHash(ctx context.Context, raw string) (string, error) {
	hashID, err := s.ResolveHashID(ctx, raw)
	if err != nil {
		return "", err
	}
	if _, err := s.GetByHashID(ctx, hashID); err != nil {
		return "", err
	}
	return hashID, nil
}
