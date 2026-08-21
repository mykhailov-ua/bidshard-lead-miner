package sink

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

const (
	ExportFormatNDJSON = "ndjson"
	ExportFormatPretty = "pretty"
)

// LeadExport returns the canonical lead document for JSON export (matches Mongo schema).
func LeadExport(lead model.Lead) LeadDoc {
	return ToLeadDoc(lead)
}

func ResolveExportFormat(path, format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "auto":
		if i := strings.LastIndex(path, "."); i >= 0 {
			ext := strings.ToLower(path[i:])
			if strings.HasSuffix(ext, ".json") && !strings.HasSuffix(ext, ".jsonl") && !strings.HasSuffix(ext, ".ndjson") {
				return ExportFormatPretty
			}
		}
		return ExportFormatNDJSON
	case "json", "json-pretty", "pretty":
		return ExportFormatPretty
	case "ndjson", "jsonl":
		return ExportFormatNDJSON
	default:
		return strings.ToLower(format)
	}
}

func EncodeLeadExport(w io.Writer, lead model.Lead, format string) error {
	doc := LeadExport(lead)
	switch format {
	case ExportFormatPretty:
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		_, err = w.Write([]byte("\n\n"))
		return err
	default:
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(doc)
	}
}
