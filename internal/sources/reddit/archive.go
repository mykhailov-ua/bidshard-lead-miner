package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/validate"
)

const (
	defaultArchiveSince   = "2025-01-01"
	defaultArchiveOut     = "data/export/reddit_archive.ndjson"
	defaultRequestPause   = 2 * time.Second
	defaultArchivePageSz  = 100
	defaultHTTPTimeout    = 45 * time.Second
	maxBackoff            = 120 * time.Second
	initialBackoff        = 4 * time.Second
)

// ArchiveRow is one JSONL record for M7 offline outreach.
type ArchiveRow struct {
	Author     string  `json:"author"`
	Body       string  `json:"body"`
	CreatedUTC float64 `json:"created_utc"`
	Permalink  string  `json:"permalink"`
}

// ArchiveOptions configures the offline PullPush/Arctic Shift crawl.
type ArchiveOptions struct {
	Subreddits     []string
	Since          time.Time
	Until          time.Time
	Out            string
	Filter         bool
	RequestPause   time.Duration
	BackoffInitial time.Duration
	BackoffMax     time.Duration
	PageSize       int
	HTTPClient     *http.Client
	PullPushSub    string
	PullPushCmt    string
	ArcticShift    string
}

// ArchiveResult summarizes one offline run.
type ArchiveResult struct {
	Fetched  int
	Written  int
	Filtered int
}

// PassesArchiveFilter keeps buyer voice or tracker+pain (M3/M8 telegram gate family).
func PassesArchiveFilter(text string) bool {
	return validate.HasTrackerPainMessage(text) ||
		validate.HasCommercialPainIntent(text) ||
		validate.HasBuyerQuestionPattern(text) ||
		scoring.HasBuyerIntentSignal(text)
}

// ArchiveOptionsFromConfig builds defaults from env-backed config.
func ArchiveOptionsFromConfig(cfg config.Config) ArchiveOptions {
	since, _ := time.Parse("2006-01-02", defaultArchiveSince)
	return ArchiveOptions{
		Subreddits:   cfg.RedditSubreddits,
		Since:        since.UTC(),
		Until:        time.Now().UTC(),
		Out:          defaultArchiveOut,
		Filter:       true,
		RequestPause: defaultRequestPause,
		PageSize:     defaultArchivePageSz,
		PullPushSub:  pullPushSubmissions,
		PullPushCmt:  pullPushComments,
		ArcticShift:  arcticShiftSubmissions,
	}
}

// RunArchive fetches subreddit history and writes filtered JSONL.
func RunArchive(ctx context.Context, opts ArchiveOptions) (ArchiveResult, error) {
	if len(opts.Subreddits) == 0 {
		return ArchiveResult{}, fmt.Errorf("reddit archive: no subreddits configured")
	}
	if opts.Since.IsZero() {
		return ArchiveResult{}, fmt.Errorf("reddit archive: since date required")
	}
	if opts.Until.IsZero() {
		opts.Until = time.Now().UTC()
	}
	if opts.Out == "" {
		opts.Out = defaultArchiveOut
	}
	if opts.RequestPause <= 0 {
		opts.RequestPause = defaultRequestPause
	}
	if opts.PageSize <= 0 {
		opts.PageSize = defaultArchivePageSz
	}
	if opts.PullPushSub == "" {
		opts.PullPushSub = pullPushSubmissions
	}
	if opts.PullPushCmt == "" {
		opts.PullPushCmt = pullPushComments
	}
	if opts.ArcticShift == "" {
		opts.ArcticShift = arcticShiftSubmissions
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return ArchiveResult{}, fmt.Errorf("reddit archive mkdir: %w", err)
	}
	f, err := os.Create(opts.Out)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("reddit archive open out: %w", err)
	}
	defer func() { _ = f.Close() }()

	backoffInit := opts.BackoffInitial
	if backoffInit <= 0 {
		backoffInit = initialBackoff
	}
	archiver := &offlineArchiver{
		client:     client,
		opts:       opts,
		seen:       map[string]struct{}{},
		enc:        json.NewEncoder(f),
		backoff:    backoffInit,
		useArctic:  false,
	}

	endpoints := []struct {
		name string
		pp   string
		ar   string
	}{
		{"submission", opts.PullPushSub, opts.ArcticShift},
		{"comment", opts.PullPushCmt, strings.Replace(opts.ArcticShift, "submission", "comment", 1)},
	}

	for _, sub := range opts.Subreddits {
		for _, ep := range endpoints {
			if err := archiver.crawlEndpoint(ctx, sub, ep.name, ep.pp, ep.ar); err != nil {
				return archiver.result, err
			}
		}
	}

	slog.Info("reddit offline archive finished",
		"out", opts.Out,
		"fetched", archiver.result.Fetched,
		"written", archiver.result.Written,
		"filtered", archiver.result.Filtered,
		"subreddits", len(opts.Subreddits),
	)
	return archiver.result, nil
}

type offlineArchiver struct {
	client    *http.Client
	opts      ArchiveOptions
	seen      map[string]struct{}
	enc       *json.Encoder
	result    ArchiveResult
	backoff   time.Duration
	useArctic bool
}

func (a *offlineArchiver) crawlEndpoint(ctx context.Context, subreddit, kind, pullURL, arcticURL string) error {
	after := a.opts.Since.Unix()
	before := a.opts.Until.Unix()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		items, err := a.searchPage(ctx, subreddit, pullURL, arcticURL, after, before)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}

		minCreated := before
		for _, post := range items {
			if post.CreatedUTC > 0 && int64(post.CreatedUTC) < minCreated {
				minCreated = int64(post.CreatedUTC)
			}
			a.result.Fetched++
			row, ok := rowFromSubmission(post)
			if !ok {
				continue
			}
			key := row.Permalink
			if key == "" {
				key = fmt.Sprintf("%s:%f", row.Author, row.CreatedUTC)
			}
			if _, dup := a.seen[key]; dup {
				continue
			}
			a.seen[key] = struct{}{}

			if a.opts.Filter && !PassesArchiveFilter(row.Body) {
				a.result.Filtered++
				continue
			}

			if err := a.enc.Encode(row); err != nil {
				return fmt.Errorf("reddit archive encode: %w", err)
			}
			a.result.Written++
		}

		if minCreated <= after {
			return nil
		}
		before = minCreated - 1

		if err := sleepCtx(ctx, a.opts.RequestPause); err != nil {
			return err
		}
	}
}

func (a *offlineArchiver) searchPage(ctx context.Context, subreddit, pullURL, arcticURL string, after, before int64) ([]submission, error) {
	primary := pullURL
	fallback := arcticURL
	if a.useArctic {
		primary, fallback = arcticURL, pullURL
	}

	for attempt := 0; attempt < 8; attempt++ {
		items, status, err := a.doSearch(ctx, primary, subreddit, after, before)
		if err == nil {
			a.backoff = a.opts.BackoffInitial
			if a.backoff <= 0 {
				a.backoff = initialBackoff
			}
			return items, nil
		}
		if status == http.StatusTooManyRequests {
			wait := a.backoff
			cap := a.opts.BackoffMax
			if cap <= 0 {
				cap = maxBackoff
			}
			if wait > cap {
				wait = cap
			}
			slog.Warn("reddit archive rate limited", "subreddit", subreddit, "backoff_ms", wait.Milliseconds())
			if err := sleepCtx(ctx, wait); err != nil {
				return nil, err
			}
			a.backoff *= 2
			continue
		}
		if status > 0 && primary != fallback {
			slog.Warn("reddit archive primary failed, trying fallback", "subreddit", subreddit, "status", status)
			items, status2, err2 := a.doSearch(ctx, fallback, subreddit, after, before)
			if err2 == nil {
				a.useArctic = fallback == arcticURL
				a.backoff = a.opts.BackoffInitial
				if a.backoff <= 0 {
					a.backoff = initialBackoff
				}
				return items, nil
			}
			if status2 == http.StatusTooManyRequests {
				status = status2
				continue
			}
			return nil, err2
		}
		return nil, err
	}
	return nil, fmt.Errorf("reddit archive: exceeded retries for r/%s", subreddit)
}

func (a *offlineArchiver) doSearch(ctx context.Context, endpointURL, subreddit string, after, before int64) ([]submission, int, error) {
	params := url.Values{}
	params.Set("subreddit", subreddit)
	params.Set("size", fmt.Sprintf("%d", a.opts.PageSize))
	params.Set("sort", "desc")
	params.Set("sort_type", "created_utc")
	params.Set("after", fmt.Sprintf("%d", after))
	params.Set("before", fmt.Sprintf("%d", before))

	reqURL := strings.TrimRight(endpointURL, "/") + "/?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("reddit archive http %d", resp.StatusCode)
	}

	var parsed searchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, resp.StatusCode, err
	}
	return parsed.Data, resp.StatusCode, nil
}

func rowFromSubmission(post submission) (ArchiveRow, bool) {
	rawContent := strings.TrimSpace(post.SelfText)
	if rawContent == "" {
		rawContent = strings.TrimSpace(post.Body)
	}
	parts := make([]string, 0, 2)
	if t := strings.TrimSpace(post.Title); t != "" {
		parts = append(parts, t)
	}
	if rawContent != "" {
		parts = append(parts, rawContent)
	}
	body := strings.TrimSpace(strings.Join(parts, "\n"))
	if body == "" {
		return ArchiveRow{}, false
	}
	author := strings.TrimSpace(post.Author)
	if author == "" || strings.EqualFold(author, "[deleted]") {
		return ArchiveRow{}, false
	}
	return ArchiveRow{
		Author:     author,
		Body:       body,
		CreatedUTC: post.CreatedUTC,
		Permalink:  normalizePermalink(post.Permalink),
	}, true
}

func normalizePermalink(permalink string) string {
	p := strings.TrimSpace(permalink)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "https://www.reddit.com" + p
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
