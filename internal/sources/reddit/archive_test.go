package reddit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPassesArchiveFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"keitaro postback failing again", true},
		{"looking for voluum alternative", true},
		{"does anyone recommend a tracker for fb?", true},
		{"Weekly digest: voluum keitaro binom market news", false},
		{"good morning team", false},
	}
	for _, tc := range cases {
		if got := PassesArchiveFilter(tc.text); got != tc.want {
			t.Fatalf("text=%q got=%v want=%v", tc.text, got, tc.want)
		}
	}
}

func TestRowFromSubmission(t *testing.T) {
	t.Parallel()
	row, ok := rowFromSubmission(submission{
		Title:      "voluum alternative",
		SelfText:   "postback failing",
		Author:     "media_buyer",
		Permalink:  "/r/affiliatemarketing/comments/abc/test/",
		CreatedUTC: 1735689600,
	})
	if !ok {
		t.Fatal("expected ok")
	}
	if row.Author != "media_buyer" {
		t.Fatalf("author=%q", row.Author)
	}
	if row.Body != "voluum alternative\npostback failing" {
		t.Fatalf("body=%q", row.Body)
	}
	if row.Permalink != "https://www.reddit.com/r/affiliatemarketing/comments/abc/test/" {
		t.Fatalf("permalink=%q", row.Permalink)
	}

	_, ok = rowFromSubmission(submission{Author: "[deleted]", Body: "x"})
	if ok {
		t.Fatal("expected deleted author to drop")
	}
}

func TestNormalizePermalink(t *testing.T) {
	t.Parallel()
	if got := normalizePermalink("https://reddit.com/x"); got != "https://reddit.com/x" {
		t.Fatalf("got=%q", got)
	}
	if got := normalizePermalink("/r/test/comments/1/"); got != "https://www.reddit.com/r/test/comments/1/" {
		t.Fatalf("got=%q", got)
	}
}

func TestRunArchiveWritesFilteredJSONL(t *testing.T) {
	t.Parallel()

	page1 := map[string]any{
		"data": []map[string]any{
			{
				"id": "a1", "title": "voluum alternative", "selftext": "postback failing",
				"author": "buyer1", "permalink": "/r/affiliatemarketing/comments/a1/x/",
				"created_utc": 1736000000.0,
			},
			{
				"id": "a2", "title": "good morning", "selftext": "team standup",
				"author": "noise", "permalink": "/r/affiliatemarketing/comments/a2/x/",
				"created_utc": 1735000000.0,
			},
		},
	}
	page2 := map[string]any{"data": []map[string]any{}}

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(page1)
			return
		}
		_ = json.NewEncoder(w).Encode(page2)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "reddit.ndjson")
	since := time.Unix(1730000000, 0).UTC()
	until := time.Unix(1740000000, 0).UTC()

	res, err := RunArchive(context.Background(), ArchiveOptions{
		Subreddits:   []string{"affiliatemarketing"},
		Since:        since,
		Until:        until,
		Out:          out,
		Filter:       true,
		RequestPause: 0,
		PageSize:     100,
		HTTPClient:   srv.Client(),
		PullPushSub:  srv.URL + "/submission/",
		PullPushCmt:  srv.URL + "/comment/",
		ArcticShift:  srv.URL + "/arctic/submission/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 {
		t.Fatalf("written=%d want 1 (filtered=%d fetched=%d)", res.Written, res.Filtered, res.Fetched)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var row ArchiveRow
	if err := json.Unmarshal(raw[:len(raw)-1], &row); err != nil {
		t.Fatalf("decode: %v raw=%q", err, string(raw))
	}
	if row.Author != "buyer1" {
		t.Fatalf("author=%q", row.Author)
	}
	if row.Body == "" || row.Permalink == "" || row.CreatedUTC == 0 {
		t.Fatalf("row=%+v", row)
	}
}

func TestRunArchiveBackoffOn429(t *testing.T) {
	t.Parallel()

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if attempts > 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id": "b1", "title": "need tracker", "selftext": "postback failing",
					"author": "buyer2", "permalink": "/r/adops/comments/b1/x/",
					"created_utc": 1736000000.0,
				},
			},
		})
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "reddit.ndjson")
	res, err := RunArchive(context.Background(), ArchiveOptions{
		Subreddits:     []string{"adops"},
		Since:          time.Unix(1730000000, 0).UTC(),
		Until:          time.Unix(1740000000, 0).UTC(),
		Out:            out,
		Filter:         true,
		RequestPause:   0,
		BackoffInitial: 10 * time.Millisecond,
		BackoffMax:     50 * time.Millisecond,
		PageSize:       100,
		HTTPClient:     srv.Client(),
		PullPushSub:    srv.URL + "/submission/",
		PullPushCmt:    srv.URL + "/comment/",
		ArcticShift:    srv.URL + "/arctic/submission/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatalf("attempts=%d want >=2", attempts)
	}
	if res.Written != 1 {
		t.Fatalf("written=%d", res.Written)
	}
}

func TestRunArchiveArcticFallback(t *testing.T) {
	t.Parallel()

	okPages := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "pullpush") {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		okPages++
		if okPages > 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id": "c1", "title": "binom migration", "selftext": "postback broken",
					"author": "buyer3", "permalink": "/r/media_buying/comments/c1/x/",
					"created_utc": 1736000000.0,
				},
			},
		})
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "reddit.ndjson")
	res, err := RunArchive(context.Background(), ArchiveOptions{
		Subreddits:   []string{"media_buying"},
		Since:        time.Unix(1730000000, 0).UTC(),
		Until:        time.Unix(1740000000, 0).UTC(),
		Out:          out,
		Filter:       true,
		RequestPause: 0,
		PageSize:     100,
		HTTPClient:   srv.Client(),
		PullPushSub:  srv.URL + "/pullpush/submission/",
		PullPushCmt:  srv.URL + "/pullpush/comment/",
		ArcticShift:  srv.URL + "/arctic/submission/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Written != 1 {
		t.Fatalf("written=%d", res.Written)
	}
}
