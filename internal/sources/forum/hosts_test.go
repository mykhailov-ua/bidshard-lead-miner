package forum

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultForumHostsIncludeLongTail(t *testing.T) {
	ConfigureHosts(nil)
	for _, host := range []string{
		"gpwa.org",
		"affroom.com",
		"cpaelites.com",
		"digitalpoint.com",
	} {
		if !IsKnownForumHost(host) {
			t.Fatalf("expected default host %q", host)
		}
		if !IsKnownForumHost("www." + host) {
			t.Fatalf("expected www.%s", host)
		}
	}
}

func TestForumHostAllowlistExtraAndSubdomain(t *testing.T) {
	ConfigureHosts([]string{"custom-forum.test"})
	if !IsKnownForumHost("custom-forum.test") {
		t.Fatal("expected extra host")
	}
	if !IsKnownForumHost("forums.gpwa.org") {
		t.Fatal("expected subdomain match for gpwa.org")
	}
	if IsKnownForumHost("not-a-forum.example") {
		t.Fatal("unexpected host match")
	}
}

func TestLoadHostAllowlistFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.txt")
	content := "# comment\n\ncpamafia.co\n  affportal.test  \n# tail\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHostAllowlistFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "cpamafia.co" || got[1] != "affportal.test" {
		t.Fatalf("hosts=%v", got)
	}
	ConfigureHosts(got)
	if !IsKnownForumHost("cpamafia.co") {
		t.Fatal("expected file host")
	}
}

func TestIsForumThreadURLPatterns(t *testing.T) {
	ConfigureHosts(nil)
	cases := []struct {
		url  string
		want bool
	}{
		{"https://www.affiliatefix.com/threads/voluum-alt.1001/", true},
		{"https://afflift.com/t/tracker-pain.42/", true},
		{"https://forums.gpwa.org/threads/migration-thread.500/", true},
		{"https://www.cpaelites.com/threads/postback-issue.99/", true},
		{"https://digitalpoint.com/forums/showthread.php?t=12345", true},
		{"https://blackhatworld.com/seo/voluum-alternative.123/", true},
		{"https://affiliatefix.com/forums/list/", false},
		{"https://example.com/threads/foo.1/", false},
	}
	for _, tc := range cases {
		if got := IsForumThreadURL(tc.url); got != tc.want {
			t.Fatalf("url=%q got=%v want=%v", tc.url, got, tc.want)
		}
	}
}

func TestAppendThreadDiscoveriesNewHost(t *testing.T) {
	ConfigureHosts(nil)
	dir := t.TempDir()
	path := filepath.Join(dir, "discovered_forum_threads.json")
	added, err := AppendThreadDiscoveries(path, "serp", "site:gpwa.org tracker", []ThreadDiscovery{
		{
			URL:     "https://forums.gpwa.org/threads/keitaro-migration.9001/",
			Title:   "Keitaro migration pain",
			Snippet: "postback failing",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	reg, err := LoadThreadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Threads) != 1 || reg.Threads[0].Host != "forums.gpwa.org" {
		t.Fatalf("threads=%v", reg.Threads)
	}
}
