package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TelegramPersonDoc is OSINT person registry (sidecar JSON sync).
type TelegramPersonDoc struct {
	UserID         int64     `bson:"user_id" json:"user_id"`
	PersonKey      string    `bson:"person_key" json:"person_key"`
	PersonUsername string    `bson:"person_username" json:"person_username"`
	OutreachFit    string    `bson:"outreach_fit" json:"outreach_fit"`
	SourceChat     string    `bson:"source_chat" json:"source_chat"`
	SourceChatRef  string    `bson:"source_chat_ref,omitempty" json:"source_chat_ref,omitempty"`
	DiscoveredVia  string    `bson:"discovered_via" json:"discovered_via"`
	ProfileBio     string    `bson:"profile_bio,omitempty" json:"profile_bio,omitempty"`
	MemberChats    []string  `bson:"member_chats,omitempty" json:"member_chats,omitempty"`
	MemberChatRefs []string  `bson:"member_chat_refs,omitempty" json:"member_chat_refs,omitempty"`
	MemberEdges    []string  `bson:"member_edges,omitempty" json:"member_edges,omitempty"`
	UpdatedAt      time.Time `bson:"updated_at" json:"updated_at"`
}

type TelegramPeopleSyncInput struct {
	UserID         int64
	PersonUsername string
	OutreachFit    string
	SourceChat     string
	SourceChatRef  string
	DiscoveredVia  string
	ProfileBio     string
	MemberChat     string
	MemberChatKey  string
}

func (s *LeadStore) SyncTelegramPeople(ctx context.Context, collName string, rows []TelegramPeopleSyncInput) (int, error) {
	if s == nil || s.leads == nil {
		return 0, fmt.Errorf("lead store not initialized")
	}
	if collName == "" {
		collName = "telegram_people"
	}
	coll := s.leads.Database().Collection(collName)
	if len(rows) == 0 {
		return 0, nil
	}
	now := time.Now().UTC()
	upserted := 0
	writeCtx, cancel := context.WithTimeout(ctx, s.writeTimeout)
	defer cancel()

	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		username := strings.TrimSpace(strings.TrimPrefix(row.PersonUsername, "@"))
		personKey := TelegramPersonKey(row.UserID)
		set := bson.M{
			"user_id":         row.UserID,
			"person_key":      personKey,
			"person_username": username,
			"outreach_fit":    strings.TrimSpace(row.OutreachFit),
			"updated_at":      now,
		}
		if bio := strings.TrimSpace(row.ProfileBio); bio != "" {
			set["profile_bio"] = bio
		}
		if chat := strings.TrimSpace(row.SourceChat); chat != "" {
			set["source_chat"] = strings.ToLower(strings.TrimPrefix(chat, "@"))
		}
		sourceRef := strings.TrimSpace(row.SourceChatRef)
		if sourceRef == "" {
			sourceRef = TelegramChatRefFromUsername(row.SourceChat)
		}
		if sourceRef != "" {
			set["source_chat_ref"] = sourceRef
		}
		if via := strings.TrimSpace(row.DiscoveredVia); via != "" {
			set["discovered_via"] = via
		}
		addToSet := bson.M{}
		if mc := strings.TrimSpace(row.MemberChat); mc != "" {
			addToSet["member_chats"] = strings.ToLower(strings.TrimPrefix(mc, "@"))
		}
		memberRef := TelegramChatRef(row.MemberChatKey)
		if memberRef == "" {
			memberRef = TelegramChatRefFromUsername(row.MemberChat)
		}
		if memberRef != "" {
			addToSet["member_chat_refs"] = memberRef
			if edge := TelegramMemberEdgeKey(memberRef, personKey); edge != "" {
				addToSet["member_edges"] = edge
			}
		}
		update := bson.M{"$set": set}
		if len(addToSet) > 0 {
			update["$addToSet"] = addToSet
		}
		res, err := coll.UpdateOne(
			writeCtx,
			bson.M{"user_id": row.UserID},
			update,
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return upserted, err
		}
		if res.UpsertedCount > 0 || res.ModifiedCount > 0 {
			upserted++
		}
	}
	return upserted, nil
}

func (s *LeadStore) ExportTelegramPeopleNDJSON(ctx context.Context, collName string, limit int64) (ExportResult, error) {
	if s == nil || s.leads == nil {
		return ExportResult{}, fmt.Errorf("lead store not initialized")
	}
	if collName == "" {
		collName = "telegram_people"
	}
	if limit <= 0 || limit > s.exportMaxRows {
		limit = s.exportMaxRows
	}
	coll := s.leads.Database().Collection(collName)
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	cur, err := coll.Find(
		queryCtx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return ExportResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	file, err := os.CreateTemp("", "telegram-people-*.ndjson")
	if err != nil {
		return ExportResult{}, fmt.Errorf("create temp export file: %w", err)
	}
	path := file.Name()
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	rows := 0
	for cur.Next(queryCtx) {
		var doc TelegramPersonDoc
		if err := cur.Decode(&doc); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return ExportResult{}, err
		}
		FillTelegramPersonCanonicalKeys(&doc)
		if err := enc.Encode(doc); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return ExportResult{}, err
		}
		rows++
	}
	if err := cur.Err(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	return ExportResult{Path: path, Rows: rows}, nil
}
