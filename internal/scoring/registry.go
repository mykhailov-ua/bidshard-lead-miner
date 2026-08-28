package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type Registry struct {
	mu sync.RWMutex

	keywordRules    []keywordRule
	titleRules      []keywordRule
	negativeRules   []keywordRule
	hardRejectRules []keywordRule
	phraseSet       map[string]string

	highMin   int
	mediumMin int

	jsonPath string
}

func NewRegistry(jsonPath string) *Registry {
	return &Registry{
		jsonPath:  jsonPath,
		phraseSet: map[string]string{},
		highMin:   defaultHighMin,
		mediumMin: defaultMediumMin,
	}
}

func (r *Registry) Load(ctx context.Context) error {
	return r.loadFiles(ctx, r.jsonPath, "")
}

func (r *Registry) LoadWithOverlay(ctx context.Context, basePath, overlayPath string) error {
	if overlayPath == "" {
		return r.loadFiles(ctx, basePath)
	}
	return r.loadFiles(ctx, basePath, overlayPath)
}

func (r *Registry) LoadWithOverlays(ctx context.Context, basePath string, overlayPaths ...string) error {
	return r.loadFiles(ctx, basePath, overlayPaths...)
}

func (r *Registry) loadFiles(ctx context.Context, basePath string, overlayPaths ...string) error {
	_ = ctx
	file, err := loadKeywordsFile(basePath)
	if err != nil {
		return err
	}
	for _, overlayPath := range overlayPaths {
		if strings.TrimSpace(overlayPath) == "" {
			continue
		}
		overlay, err := loadKeywordsFile(overlayPath)
		if err != nil {
			slog.Warn("keywords overlay load skipped", "path", overlayPath, "error", err)
			continue
		}
		file.Keywords = append(file.Keywords, overlay.Keywords...)
		file.Titles = append(file.Titles, overlay.Titles...)
		file.Negative = append(file.Negative, overlay.Negative...)
		file.HardReject = append(file.HardReject, overlay.HardReject...)
	}
	file = normalizeKeywordFile(file)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.keywordRules = rulesFromEntries(file.Keywords)
	r.titleRules = rulesFromEntries(file.Titles)
	r.negativeRules = rulesFromEntries(file.Negative)
	r.hardRejectRules = rulesFromEntries(file.HardReject)
	if file.Priority.HighMin > 0 {
		r.highMin = file.Priority.HighMin
	}
	if file.Priority.MediumMin > 0 {
		r.mediumMin = file.Priority.MediumMin
	}
	r.rebuildPhraseSet()

	slog.Info("keyword registry loaded",
		"keywords", len(r.keywordRules),
		"titles", len(r.titleRules),
		"negative", len(r.negativeRules),
		"hard_reject", len(r.hardRejectRules),
		"phrases", len(r.phraseSet),
	)
	return nil
}

func (r *Registry) rebuildPhraseSet() {
	r.phraseSet = map[string]string{}
	for _, rule := range r.keywordRules {
		r.phraseSet[NormalizePhrase(rule.phrase)] = rule.id
	}
	for _, rule := range r.titleRules {
		r.phraseSet[NormalizePhrase(rule.phrase)] = rule.id
	}
	for _, rule := range r.negativeRules {
		r.phraseSet[NormalizePhrase(rule.phrase)] = rule.id
	}
}

type HardRejectHit struct {
	ID     string
	Phrase string
}

func (r *Registry) HardReject(text string) (HardRejectHit, bool) {
	if r == nil {
		return HardRejectHit{}, false
	}
	r.mu.RLock()
	rules := append([]keywordRule(nil), r.hardRejectRules...)
	r.mu.RUnlock()

	body := strings.ToLower(text)
	for _, rule := range rules {
		phrase := rule.phrase
		if phrase == "" {
			continue
		}
		if strings.Contains(body, phrase) {
			return HardRejectHit{ID: rule.id, Phrase: rule.phrase}, true
		}
	}
	return HardRejectHit{}, false
}

func (r *Registry) Snapshot() (kw, title, neg []keywordRule, highMin, mediumMin int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]keywordRule{}, r.keywordRules...),
		append([]keywordRule{}, r.titleRules...),
		append([]keywordRule{}, r.negativeRules...),
		r.highMin, r.mediumMin
}

// KeywordWeights returns current keyword id -> weight for stats tuning reports.
func (r *Registry) KeywordWeights() map[string]int {
	if r == nil {
		return map[string]int{}
	}
	kw, _, _, _, _ := r.Snapshot()
	out := make(map[string]int, len(kw))
	for _, rule := range kw {
		if rule.id == "" {
			continue
		}
		out[rule.id] = rule.weight
	}
	return out
}

func (r *Registry) Prescan(text string) bool {
	return AnalyzeWithRegistry(r, text).Score > 0
}

func loadKeywordsFile(path string) (*keywordsFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keywords json: %w", err)
	}
	var file keywordsFile
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("parse keywords json: %w", err)
	}
	return &file, nil
}

func NormalizePhrase(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func normalizeKeywordFile(file *keywordsFile) *keywordsFile {
	if file == nil {
		return &keywordsFile{}
	}
	kw, kwHard := splitHardRejectEntries(file.Keywords)
	file.Keywords = kw
	file.HardReject = append(file.HardReject, kwHard...)

	titles, titlesHard := splitHardRejectEntries(file.Titles)
	file.Titles = titles
	file.HardReject = append(file.HardReject, titlesHard...)

	neg, negHard := splitHardRejectEntries(file.Negative)
	file.Negative = neg
	file.HardReject = append(file.HardReject, negHard...)
	return file
}

func splitHardRejectEntries(entries []keywordEntry) (score []keywordEntry, hard []keywordEntry) {
	for _, e := range entries {
		if isHardRejectTag(e.Tag) {
			hard = append(hard, e)
			continue
		}
		score = append(score, e)
	}
	return score, hard
}

func isHardRejectTag(tag string) bool {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "hard-reject", "hard_reject":
		return true
	default:
		return false
	}
}
