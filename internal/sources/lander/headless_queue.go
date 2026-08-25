package lander

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/metrics"
)

const DefaultHeadlessQueuePath = "data/runtime/headless_queue.json"

// HeadlessQueueItem is one deferred Playwright fetch.
type HeadlessQueueItem struct {
	URL          string `json:"url"`
	SourceFamily string `json:"source_family"`
	Reason       string `json:"reason"`
	QueuedAt     string `json:"queued_at"`
	Attempts     int    `json:"attempts"`
}

type headlessQueueFile struct {
	Items []HeadlessQueueItem `json:"items"`
}

var headlessQueueMu sync.Mutex

// EnqueueHeadless adds a URL to the nightly headless drain queue (deduped).
func EnqueueHeadless(path string, item HeadlessQueueItem) error {
	if path == "" {
		path = DefaultHeadlessQueuePath
	}
	item.URL = strings.TrimSpace(item.URL)
	if item.URL == "" {
		return fmt.Errorf("headless queue url empty")
	}
	if item.SourceFamily == "" {
		item.SourceFamily = "lander"
	}
	if item.Reason == "" {
		item.Reason = "needs_headless"
	}
	if item.QueuedAt == "" {
		item.QueuedAt = time.Now().UTC().Format(time.RFC3339)
	}

	headlessQueueMu.Lock()
	defer headlessQueueMu.Unlock()

	file, err := readHeadlessQueue(path)
	if err != nil {
		return err
	}
	key := strings.ToLower(item.URL)
	for _, existing := range file.Items {
		if strings.ToLower(existing.URL) == key {
			return nil
		}
	}
	file.Items = append(file.Items, item)
	metrics.RecordHeadlessQueued(1)
	return writeHeadlessQueue(path, file)
}

// LoadPendingHeadless returns queued items ready for drain (attempts < maxAttempts).
func LoadPendingHeadless(path string, limit, maxAttempts int) ([]HeadlessQueueItem, error) {
	if path == "" {
		path = DefaultHeadlessQueuePath
	}
	if limit <= 0 {
		limit = 25
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	headlessQueueMu.Lock()
	defer headlessQueueMu.Unlock()

	file, err := readHeadlessQueue(path)
	if err != nil {
		return nil, err
	}
	var pending []HeadlessQueueItem
	for _, item := range file.Items {
		if item.Attempts >= maxAttempts {
			continue
		}
		pending = append(pending, item)
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].QueuedAt < pending[j].QueuedAt
	})
	if len(pending) > limit {
		pending = pending[:limit]
	}
	return pending, nil
}

// RemoveHeadlessQueueItem deletes a URL from the queue after successful drain.
func RemoveHeadlessQueueItem(path, rawURL string) error {
	if path == "" {
		path = DefaultHeadlessQueuePath
	}
	key := strings.ToLower(strings.TrimSpace(rawURL))
	if key == "" {
		return nil
	}

	headlessQueueMu.Lock()
	defer headlessQueueMu.Unlock()

	file, err := readHeadlessQueue(path)
	if err != nil {
		return err
	}
	out := file.Items[:0]
	for _, item := range file.Items {
		if strings.ToLower(item.URL) == key {
			continue
		}
		out = append(out, item)
	}
	file.Items = out
	return writeHeadlessQueue(path, file)
}

// BumpHeadlessQueueAttempts increments drain failure count for a URL.
func BumpHeadlessQueueAttempts(path, rawURL string) error {
	if path == "" {
		path = DefaultHeadlessQueuePath
	}
	key := strings.ToLower(strings.TrimSpace(rawURL))
	if key == "" {
		return nil
	}

	headlessQueueMu.Lock()
	defer headlessQueueMu.Unlock()

	file, err := readHeadlessQueue(path)
	if err != nil {
		return err
	}
	for i := range file.Items {
		if strings.ToLower(file.Items[i].URL) == key {
			file.Items[i].Attempts++
			return writeHeadlessQueue(path, file)
		}
	}
	return nil
}

// TrimHeadlessQueue drops oldest items when over maxSize.
func TrimHeadlessQueue(path string, maxSize int) error {
	if maxSize <= 0 {
		return nil
	}
	headlessQueueMu.Lock()
	defer headlessQueueMu.Unlock()

	file, err := readHeadlessQueue(path)
	if err != nil {
		return err
	}
	if len(file.Items) <= maxSize {
		return nil
	}
	sort.Slice(file.Items, func(i, j int) bool {
		return file.Items[i].QueuedAt < file.Items[j].QueuedAt
	})
	file.Items = file.Items[len(file.Items)-maxSize:]
	return writeHeadlessQueue(path, file)
}

// CountHeadlessQueue returns queued URLs waiting for nightly drain.
func CountHeadlessQueue(path string) (int, error) {
	if path == "" {
		path = DefaultHeadlessQueuePath
	}
	file, err := readHeadlessQueue(path)
	if err != nil {
		return 0, err
	}
	return len(file.Items), nil
}

func readHeadlessQueue(path string) (headlessQueueFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return headlessQueueFile{Items: []HeadlessQueueItem{}}, nil
		}
		return headlessQueueFile{}, err
	}
	var file headlessQueueFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return headlessQueueFile{}, err
	}
	if file.Items == nil {
		file.Items = []HeadlessQueueItem{}
	}
	return file, nil
}

func writeHeadlessQueue(path string, file headlessQueueFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// RawSourceLabel builds a model.RawItem source label from a queue item.
func RawSourceLabel(item HeadlessQueueItem) string {
	host := hostFromURL(item.URL)
	switch strings.ToLower(strings.TrimSpace(item.SourceFamily)) {
	case "tgweb":
		if host == "" {
			return "tgweb"
		}
		return "tgweb:" + host
	default:
		if host == "" {
			return "lander"
		}
		return "lander:" + host
	}
}
