package sources

import (
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sourcedisable"
)

func TestBuildSkipsDisabledSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/disabled.json"
	if err := sourcedisable.Save(path, []string{"forum"}, []string{"test"}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Source:              "forum,reddit",
		DisabledSourcesPath: path,
		ForumSeedPath:       "data/seeds/forum_threads.csv",
	}
	srcs := Build(cfg)
	names := make(map[string]struct{})
	for _, s := range srcs {
		names[s.Name()] = struct{}{}
	}
	if _, ok := names["forum"]; ok {
		t.Fatal("forum should be skipped")
	}
	if _, ok := names["reddit"]; !ok {
		t.Fatal("reddit should remain")
	}
}
