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

	keywordRules  []keywordRule
	titleRules    []keywordRule
	negativeRules []keywordRule
	phraseSet     map[string]string

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
	return r.loadFiles(ctx, basePath, overlayPath)
}

func (r *Registry) loadFiles(ctx context.Context, basePath, overlayPath string) error {
	_ = ctx
	file, err := loadKeywordsFile(basePath)
	if err != nil {
		return err
	}
	if overlayPath != "" {
		overlay, err := loadKeywordsFile(overlayPath)
		if err != nil {
			return err
		}
		file.Keywords = append(file.Keywords, overlay.Keywords...)
		file.Titles = append(file.Titles, overlay.Titles...)
		file.Negative = append(file.Negative, overlay.Negative...)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.keywordRules = rulesFromEntries(file.Keywords)
	r.titleRules = rulesFromEntries(file.Titles)
	r.negativeRules = rulesFromEntries(file.Negative)
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

func (r *Registry) Snapshot() (kw, title, neg []keywordRule, highMin, mediumMin int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]keywordRule{}, r.keywordRules...),
		append([]keywordRule{}, r.titleRules...),
		append([]keywordRule{}, r.negativeRules...),
		r.highMin, r.mediumMin
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
