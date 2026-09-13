package ops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const GlobalSettingsID = "global"

// ErrSettingsNotConfigured is returned when crm_settings collection is not wired.
var ErrSettingsNotConfigured = fmt.Errorf("crm settings store not configured")

// GlobalSettings are ops toggles shared by parser and crm-bot (Mongo document _id=global).
type GlobalSettings struct {
	LLMScoringEnabled bool      `bson:"llm_scoring_enabled"`
	UpdatedAt         time.Time `bson:"updated_at"`
}

// SettingsStore reads/writes crm_settings with a short TTL cache for the hot accept path.
type SettingsStore struct {
	coll *mongo.Collection
	ttl  time.Duration

	mu       sync.RWMutex
	cached   GlobalSettings
	cachedAt time.Time
}

func ConnectSettings(ctx context.Context, client *mongo.Client, dbName, collection string) (*SettingsStore, error) {
	if client == nil {
		return nil, fmt.Errorf("mongo client nil")
	}
	collection = collectionOrDefault(collection)
	coll := client.Database(dbName).Collection(collection)
	s := &SettingsStore{coll: coll, ttl: 15 * time.Second}
	_, _ = s.Get(ctx)
	return s, nil
}

func collectionOrDefault(collection string) string {
	if collection = trim(collection); collection == "" {
		return "crm_settings"
	}
	return collection
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// LLMScoringEnabled defaults true when Mongo is unreachable or document missing.
func (s *SettingsStore) LLMScoringEnabled(ctx context.Context) bool {
	if s == nil || s.coll == nil {
		return true
	}
	s.mu.RLock()
	if time.Since(s.cachedAt) < s.ttl {
		enabled := s.cached.LLMScoringEnabled
		s.mu.RUnlock()
		return enabled
	}
	s.mu.RUnlock()

	doc, err := s.Get(ctx)
	if err != nil {
		return true
	}
	return doc.LLMScoringEnabled
}

func (s *SettingsStore) Get(ctx context.Context) (GlobalSettings, error) {
	if s == nil || s.coll == nil {
		return defaultSettings(), nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var doc GlobalSettings
	err := s.coll.FindOne(readCtx, bson.M{"_id": GlobalSettingsID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		doc = defaultSettings()
		s.storeCache(doc)
		return doc, nil
	}
	if err != nil {
		return defaultSettings(), err
	}
	s.storeCache(doc)
	return doc, nil
}

func (s *SettingsStore) SetLLMScoringEnabled(ctx context.Context, enabled bool) error {
	if s == nil || s.coll == nil {
		return fmt.Errorf("settings store not configured")
	}
	now := time.Now().UTC()
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.coll.UpdateOne(
		writeCtx,
		bson.M{"_id": GlobalSettingsID},
		bson.M{
			"$set": bson.M{
				"llm_scoring_enabled": enabled,
				"updated_at":          now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}
	s.storeCache(GlobalSettings{LLMScoringEnabled: enabled, UpdatedAt: now})
	return nil
}

func (s *SettingsStore) storeCache(doc GlobalSettings) {
	s.mu.Lock()
	s.cached = doc
	s.cachedAt = time.Now()
	s.mu.Unlock()
}

func defaultSettings() GlobalSettings {
	return GlobalSettings{LLMScoringEnabled: true}
}
