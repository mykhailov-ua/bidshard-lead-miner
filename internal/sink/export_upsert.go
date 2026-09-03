package sink

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

func upsertLeadExport(path string, lead model.Lead, format string) error {
	leads, order, err := readLeadExportFile(path, format)
	if err != nil {
		return err
	}
	if _, exists := leads[lead.HashID]; !exists {
		order = append(order, lead.HashID)
	}
	leads[lead.HashID] = lead
	return writeLeadExportFile(path, order, leads, format)
}

func readLeadExportFile(path, format string) (map[string]model.Lead, []string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]model.Lead{}, nil, nil
		}
		return nil, nil, err
	}
	leads := map[string]model.Lead{}
	var order []string
	switch format {
	case ExportFormatPretty:
		for _, chunk := range strings.Split(string(raw), "\n\n") {
			chunk = strings.TrimSpace(chunk)
			if chunk == "" {
				continue
			}
			var doc LeadDoc
			if err := json.Unmarshal([]byte(chunk), &doc); err != nil {
				return nil, nil, err
			}
			if doc.HashID == "" {
				continue
			}
			if _, ok := leads[doc.HashID]; !ok {
				order = append(order, doc.HashID)
			}
			leads[doc.HashID] = LeadDocToModel(doc)
		}
	default:
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var doc LeadDoc
			if err := json.Unmarshal([]byte(line), &doc); err != nil {
				return nil, nil, err
			}
			if doc.HashID == "" {
				continue
			}
			if _, ok := leads[doc.HashID]; !ok {
				order = append(order, doc.HashID)
			}
			leads[doc.HashID] = LeadDocToModel(doc)
		}
	}
	return leads, order, nil
}

func writeLeadExportFile(path string, order []string, leads map[string]model.Lead, format string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	var buf bytes.Buffer
	for _, hashID := range order {
		lead, ok := leads[hashID]
		if !ok {
			continue
		}
		if err := EncodeLeadExport(&buf, lead, format); err != nil {
			return err
		}
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, buf.Bytes(), mode)
}
