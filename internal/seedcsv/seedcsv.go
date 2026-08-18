package seedcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// ReadRecords parses a seed CSV file, skipping blank lines and whole-line comments (#).
func ReadRecords(path string) ([][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var filtered strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		filtered.WriteString(line)
		filtered.WriteByte('\n')
	}

	reader := csv.NewReader(strings.NewReader(filtered.String()))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return records, nil
}
