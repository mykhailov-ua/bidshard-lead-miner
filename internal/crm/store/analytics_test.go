package store

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestDecodeStatusFunnel(t *testing.T) {
	raw := bson.A{
		bson.M{"_id": "new", "count": int32(5)},
		bson.M{"_id": "", "count": int64(2)},
	}
	got := decodeStatusFunnel(raw)
	if len(got) != 2 || got[0].Status != "new" || got[0].Count != 5 {
		t.Fatalf("got=%+v", got)
	}
	if got[1].Status != "unknown" {
		t.Fatalf("empty status mapped to %q", got[1].Status)
	}
}

func TestDecodeCount(t *testing.T) {
	raw := bson.A{bson.M{"n": int32(7)}}
	if got := decodeCount(raw); got != 7 {
		t.Fatalf("count=%d", got)
	}
}

func TestDecodeTopSources(t *testing.T) {
	raw := bson.A{bson.M{"_id": "forum:test", "count": int64(3)}}
	got := decodeTopSources(raw)
	if len(got) != 1 || got[0].Source != "forum:test" || got[0].Count != 3 {
		t.Fatalf("got=%+v", got)
	}
}
