package serp

import "testing"

func TestSelectSerpDorksRotatesAndCaps(t *testing.T) {
	t.Parallel()

	in := []string{"a", "b", "c", "d", "e"}
	got := selectSerpDorks(in, 2, 0, 2)
	if len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Fatalf("rotate+max got=%v", got)
	}
	got = selectSerpDorks(in, 0, 3, 0)
	if len(got) != 3 || got[0] != "a" {
		t.Fatalf("batch got=%v", got)
	}
}

func TestLimitSerpDorks(t *testing.T) {
	t.Parallel()

	in := []string{"a", "b", "c"}
	if got := limitSerpDorks(in, 0); len(got) != 3 {
		t.Fatalf("max=0 got=%v", got)
	}
	if got := limitSerpDorks(in, 2); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("max=2 got=%v", got)
	}
}

func TestSerpHarvestTelegramDorksExcludesJobBoards(t *testing.T) {
	t.Parallel()

	got := serpHarvestTelegramDorks([]string{
		`site:t.me voluum alternative`,
		`site:jobs.dou.ua media buyer keitaro`,
		`site:djinni.co media buyer gambling`,
		`site:affiliatefix.com t.me tracker`,
	})
	if len(got) != 2 {
		t.Fatalf("got=%d dorks want 2: %v", len(got), got)
	}
	if got[0] != `site:t.me voluum alternative` {
		t.Fatalf("first=%q", got[0])
	}
	if got[1] != `site:affiliatefix.com t.me tracker` {
		t.Fatalf("second=%q", got[1])
	}
}
