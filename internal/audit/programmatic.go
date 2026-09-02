package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/bidshard/parser/internal/filter"
)

// ProgrammaticReport summarizes would_drop programmatic vertical on an export corpus.
type ProgrammaticReport struct {
	Total      int
	WouldDrop  int
	BySource   map[string]int
	ByReason   map[string]int
}

type leadRow struct {
	Source  string `json:"source"`
	Snippet string `json:"snippet"`
	Text    string `json:"text"`
	Title   string `json:"title"`
}

// ScanLeadsJSONL reads NDJSON/JSONL export rows and applies RejectProgrammaticContext.
func ScanLeadsJSONL(path string) (ProgrammaticReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return ProgrammaticReport{}, err
	}
	defer f.Close()
	return ScanLeadsJSONLReader(f)
}

// ScanLeadsJSONLReader scans export rows from r (one JSON object per line).
func ScanLeadsJSONLReader(r io.Reader) (ProgrammaticReport, error) {
	rep := ProgrammaticReport{
		BySource: map[string]int{},
		ByReason: map[string]int{},
	}
	sc := bufio.NewScanner(r)
	const maxLine = 4 << 20
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, maxLine)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row leadRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return rep, fmt.Errorf("parse line: %w", err)
		}
		rep.Total++
		text := strings.TrimSpace(row.Snippet)
		if text == "" {
			text = strings.TrimSpace(row.Text)
		}
		drop, reason := filter.RejectProgrammaticContext(row.Source, text, row.Title)
		if !drop {
			continue
		}
		rep.WouldDrop++
		src := strings.TrimSpace(row.Source)
		if src == "" {
			src = "unknown"
		}
		rep.BySource[src]++
		if reason == "" {
			reason = "programmatic"
		}
		rep.ByReason[reason]++
	}
	if err := sc.Err(); err != nil {
		return rep, err
	}
	return rep, nil
}

// FormatProgrammaticReport prints operator-safe counts (no PII bodies).
func FormatProgrammaticReport(rep ProgrammaticReport) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "total=%d would_drop_programmatic=%d\n", rep.Total, rep.WouldDrop)
	if len(rep.ByReason) > 0 {
		keys := sortedKeys(rep.ByReason)
		_, _ = fmt.Fprintf(&b, "by_reason:\n")
		for _, k := range keys {
			_, _ = fmt.Fprintf(&b, "  %s: %d\n", k, rep.ByReason[k])
		}
	}
	if len(rep.BySource) > 0 {
		keys := sortedKeys(rep.BySource)
		_, _ = fmt.Fprintf(&b, "by_source:\n")
		for _, k := range keys {
			_, _ = fmt.Fprintf(&b, "  %s: %d\n", k, rep.BySource[k])
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
