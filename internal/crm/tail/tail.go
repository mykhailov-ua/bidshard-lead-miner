package tail

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Options struct {
	MinScore int
	JSON     bool
}

type Emitter func(sink.LeadDoc)

// WatchMongo prints new lead inserts from a Mongo change stream.
func WatchMongo(ctx context.Context, client *mongo.Client, dbName, collName string, opts Options, emit Emitter) error {
	if client == nil {
		return fmt.Errorf("mongo client nil")
	}
	if emit == nil {
		emit = func(doc sink.LeadDoc) {}
	}

	coll := client.Database(dbName).Collection(collName)
	stream, err := coll.Watch(ctx, mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"operationType": "insert"}}},
	}, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		return fmt.Errorf("change stream: %w", err)
	}
	defer func() { _ = stream.Close(ctx) }()

	for stream.Next(ctx) {
		var ev struct {
			FullDocument sink.LeadDoc `bson:"fullDocument"`
		}
		if err := stream.Decode(&ev); err != nil {
			return fmt.Errorf("decode change event: %w", err)
		}
		if opts.MinScore > 0 && ev.FullDocument.Score < opts.MinScore {
			continue
		}
		emit(ev.FullDocument)
	}
	if err := stream.Err(); err != nil && ctx.Err() == nil {
		return err
	}
	return ctx.Err()
}

// FollowJSONL tails an append-only NDJSON export. path "-" reads stdin (for ssh pipes).
func FollowJSONL(ctx context.Context, path string, opts Options, emit Emitter) error {
	if emit == nil {
		emit = func(doc sink.LeadDoc) {}
	}
	if path == "-" {
		return followJSONLReader(ctx, bufio.NewReader(os.Stdin), opts, emit)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return followJSONLReader(ctx, bufio.NewReader(f), opts, emit)
}

func followJSONLReader(ctx context.Context, reader *bufio.Reader, opts Options, emit Emitter) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				return err
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var doc sink.LeadDoc
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			continue
		}
		if opts.MinScore > 0 && doc.Score < opts.MinScore {
			continue
		}
		emit(doc)
	}
}

// FormatLine renders one accepted lead for terminal output.
func FormatLine(doc sink.LeadDoc) string {
	ts := doc.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	contact := formatContact(doc.Contacts)
	keywords := strings.Join(doc.Matched, ", ")
	if len(keywords) > 48 {
		keywords = keywords[:45] + "..."
	}
	snippet := strings.Join(strings.Fields(doc.Snippet), " ")
	if len(snippet) > 64 {
		snippet = snippet[:61] + "..."
	}
	geo := doc.GeoCountry
	if geo == "" {
		geo = doc.CompanyCountry
	}
	if geo == "" {
		geo = "-"
	}
	return fmt.Sprintf("[%s] score=%d %s %s geo=%s kw=%s contact=%s | %s",
		ts.Format("15:04:05"),
		doc.Score,
		doc.Priority,
		doc.Source,
		geo,
		keywords,
		contact,
		snippet,
	)
}

func formatContact(contacts []sink.StoredContact) string {
	if len(contacts) == 0 {
		return "-"
	}
	c := contacts[0]
	if c.Type != "" && c.Value != "" {
		return c.Type + ":" + c.Value
	}
	return c.Value
}
