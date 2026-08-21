package seedcheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileLooksLikeFixture(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixture := filepath.Join(dir, "forum.csv")
	if err := os.WriteFile(fixture, []byte("url\nhttps://forum-fixture.test/x/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, marker := FileLooksLikeFixture(fixture)
	if !ok || marker != "forum-fixture.test" {
		t.Fatalf("fixture=%v marker=%q", ok, marker)
	}

	live := filepath.Join(dir, "live.csv")
	if err := os.WriteFile(live, []byte("url\nhttps://affiliatefix.com/threads/a.1/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, _ := FileLooksLikeFixture(live); ok {
		t.Fatal("expected live csv not fixture")
	}
}

func TestCountDataRows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "seeds.csv")
	content := "# comment\nurl,notes\nhttps://a.test/1,a\nhttps://a.test/2,b\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := CountDataRows(path)
	if err != nil || n != 2 {
		t.Fatalf("rows=%d err=%v want 2", n, err)
	}
}

func TestProfileDefaultDev(t *testing.T) {
	t.Parallel()
	if Profile() != "dev" {
		t.Fatalf("profile=%q want dev", Profile())
	}
}
