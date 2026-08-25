package gemini

import (
	"context"
	"testing"
)

func TestRankLanderPathsReturnsSubset(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"paths":["/join-partners","/contact-us"]}`)
	candidates := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		candidates = append(candidates, "/page-"+string(rune('a'+i)))
	}
	candidates = append(candidates, "/join-partners", "/contact-us")

	got, err := cl.RankLanderPaths(context.Background(), "example.com", candidates)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected ranked paths")
	}
}
