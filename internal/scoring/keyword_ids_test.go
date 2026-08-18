package scoring

import (
	"context"
	"testing"
)

func TestKeywordIDsFromMatched(t *testing.T) {
	ctx := context.Background()
	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(ctx); err != nil {
		t.Fatal(err)
	}

	matched := []string{"voluum alternative(+25)", "postback failing(+20)"}
	ids := reg.KeywordIDsFromMatched(matched)
	if len(ids) == 0 {
		t.Fatalf("expected keyword ids, got none")
	}
}
