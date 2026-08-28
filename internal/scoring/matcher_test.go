package scoring

import (
	"context"
	"testing"
)

func TestMatchWeightedOncePerPhrase(t *testing.T) {
	t.Parallel()

	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	body := "voluum voluum voluum media buyer media buyer s2s postback"
	result := AnalyzeWithRegistry(reg, body)

	voluumHits := 0
	for _, hit := range result.Hits {
		if hit.Phrase == "voluum" {
			voluumHits++
		}
	}
	if voluumHits > 1 {
		t.Fatalf("voluum hits=%d want 1", voluumHits)
	}
	if len(result.Hits) > 8 {
		t.Fatalf("too many distinct hits=%d", len(result.Hits))
	}
}
