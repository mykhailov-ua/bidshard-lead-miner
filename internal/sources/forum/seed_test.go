package forum

import (
	"os"
	"testing"
)

func TestLoadThreadSeedsCombinedMergesWarriorCSV(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	forumCSV := dir + "/forum.csv"
	warriorCSV := dir + "/warrior.csv"
	if err := os.WriteFile(forumCSV, []byte("url\nhttps://affiliatefix.com/threads/a.1/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(warriorCSV, []byte("url\nhttps://warriorforum.com/threads/voluum-vs-keitaro.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	seeds, err := LoadThreadSeedsCombined(forumCSV, "", warriorCSV)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 2 {
		t.Fatalf("seeds=%d want 2", len(seeds))
	}
}

func TestLoadThreadURLsSkipsComments(t *testing.T) {
	t.Parallel()

	urls, err := LoadThreadURLs("../../../data/seeds/forum_threads.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) < 50 {
		t.Fatalf("urls=%d want >=50", len(urls))
	}
}
