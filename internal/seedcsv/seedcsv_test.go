package seedcsv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRecordsSkipsComments(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed.csv")
	content := `# header comment
url,notes
https://example.com/thread,voluum
# trailing comment
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := ReadRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%d want 2 (header + data)", len(records))
	}
	if records[1][0] != "https://example.com/thread" {
		t.Fatalf("url=%q", records[1][0])
	}
}
