package scoring

import (
	"context"
	"path/filepath"
	"testing"
)

func TestHardRejectRUGeoFromGrayOverlay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	reg := NewRegistry("../../data/keywords.json")
	grayPath := filepath.Join("../../data", "keywords-gray.json")
	if err := reg.LoadWithOverlay(ctx, "../../data/keywords.json", grayPath); err != nil {
		t.Fatalf("LoadWithOverlay: %v", err)
	}

	cases := []struct {
		text string
		want string
	}{
		{text: "Pay via Sberbank, voluum alternative needed", want: "sberbank"},
		{text: "Оплата руб, ищем tracker migration", want: "оплата руб"},
	}
	for _, tc := range cases {
		hit, ok := reg.HardReject(tc.text)
		if !ok {
			t.Fatalf("expected hard reject for %q", tc.text)
		}
		if hit.Phrase != tc.want {
			t.Fatalf("text=%q phrase=%q want %q", tc.text, hit.Phrase, tc.want)
		}
	}
}
