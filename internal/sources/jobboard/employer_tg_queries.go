package jobboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/entity"
)

const DefaultEmployerTGQueriesPath = "data/runtime/discovered_employer_tg_queries.json"

// EmployerTGQuery is a pending Telethon SearchRequest for an employer name.
type EmployerTGQuery struct {
	Name       string `json:"name"`
	Normalized string `json:"normalized,omitempty"`
	Source     string `json:"source,omitempty"`
	QueuedAt   string `json:"queued_at,omitempty"`
}

type EmployerTGQueryFile struct {
	Queries []EmployerTGQuery `json:"queries"`
}

func LoadEmployerTGQueries(path string) (EmployerTGQueryFile, error) {
	if path == "" {
		path = DefaultEmployerTGQueriesPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EmployerTGQueryFile{}, nil
		}
		return EmployerTGQueryFile{}, err
	}
	var f EmployerTGQueryFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return EmployerTGQueryFile{}, err
	}
	return f, nil
}

func SaveEmployerTGQueries(path string, f EmployerTGQueryFile) error {
	if path == "" {
		path = DefaultEmployerTGQueriesPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AppendEmployerTGQueries queues employer names for Telethon SearchRequest.
// Returns the number of newly queued rows.
func AppendEmployerTGQueries(path string, names []string, source string) (int, error) {
	if source == "" {
		source = "employer_reverse"
	}
	f, err := LoadEmployerTGQueries(path)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(f.Queries))
	for _, q := range f.Queries {
		key := strings.TrimSpace(q.Normalized)
		if key == "" {
			key = entity.NormalizeCompany(q.Name)
		}
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	added := 0
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		normalized := entity.NormalizeCompany(name)
		if normalized == "" || !entity.IsOrgLikeName(name) {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		f.Queries = append(f.Queries, EmployerTGQuery{
			Name:       name,
			Normalized: normalized,
			Source:     source,
			QueuedAt:   now,
		})
		added++
	}
	if added == 0 {
		return 0, nil
	}
	if err := SaveEmployerTGQueries(path, f); err != nil {
		return 0, err
	}
	return added, nil
}
