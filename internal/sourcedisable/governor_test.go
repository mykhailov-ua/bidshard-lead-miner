package sourcedisable

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/sink"
)

func TestEvaluateZeroAcceptHighRaw(t *testing.T) {
	t.Parallel()
	stats := []sink.SourceStatsDoc{
		{Source: "forum", Accepted: 0, Junk: 120},
		{Source: "reddit", Accepted: 1, Junk: 200},
		{Source: "serp", Accepted: 0, Junk: 50},
	}
	got := Evaluate(stats, 100)
	if len(got) != 1 || got[0] != "forum" {
		t.Fatalf("got=%v", got)
	}
}

func TestIsDisabledPrefixMatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "disabled.json")
	if err := Save(path, []string{"forum"}, []string{"test"}); err != nil {
		t.Fatal(err)
	}
	if !IsDisabled(path, "forum") {
		t.Fatal("forum should be disabled")
	}
	if !IsDisabled(path, "forum:affiliatefix") {
		t.Fatal("forum prefix should be disabled")
	}
	if IsDisabled(path, "reddit") {
		t.Fatal("reddit should not be disabled")
	}
	_ = os.Remove(path)
}
