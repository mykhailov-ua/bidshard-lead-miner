package sink

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubAnalysisPatcher struct {
	called bool
}

func (s *stubAnalysisPatcher) PatchLeadAnalysis(_ context.Context, _ LeadAnalysisPatch) error {
	s.called = true
	return nil
}

type stubLeadReader struct {
	doc LeadDoc
	ok  bool
}

func (s *stubLeadReader) GetLeadByHashID(_ context.Context, _ string) (LeadDoc, bool, error) {
	return s.doc, s.ok, nil
}

func TestExportSyncPatcherAppendsAfterPatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "leads.ndjson")
	export, err := NewJSONFileSink(path, ExportFormatNDJSON)
	if err != nil {
		t.Fatal(err)
	}

	reader := &stubLeadReader{
		ok: true,
		doc: LeadDoc{
			HashID:         "h1",
			TS:             time.Now().UTC(),
			AnalysisStatus: "done",
			ICP:            "starter",
			GeoCountry:     "DE",
		},
	}
	inner := &stubAnalysisPatcher{}
	patcher := NewExportSyncPatcher(inner, reader, export)

	if err := patcher.PatchLeadAnalysis(context.Background(), LeadAnalysisPatch{
		HashID:         "h1",
		AnalysisStatus: "done",
		ICP:            "starter",
		GeoCountry:     "DE",
	}); err != nil {
		t.Fatal(err)
	}
	if !inner.called {
		t.Fatal("inner patcher not called")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"hash_id":"h1"`) {
		t.Fatalf("export missing lead: %s", body)
	}
	if !strings.Contains(body, `"icp":"starter"`) {
		t.Fatalf("export missing icp: %s", body)
	}
}
