package coldpath

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
)

type stubBoostCRM struct {
	pending []sink.CrmBoostDoc
	last    struct {
		junkID string
		status string
		hash   string
	}
}

func (s *stubBoostCRM) Insert(ctx context.Context, doc sink.CrmBoostDoc) error {
	s.pending = append(s.pending, doc)
	return nil
}

func (s *stubBoostCRM) ListPending(ctx context.Context, limit int) ([]sink.CrmBoostDoc, error) {
	return append([]sink.CrmBoostDoc(nil), s.pending...), nil
}

func (s *stubBoostCRM) Resolve(ctx context.Context, junkID, status, leadHashID, why string) error {
	s.last.junkID = junkID
	s.last.status = status
	s.last.hash = leadHashID
	return nil
}

type stubBoostLeads struct {
	hash string
}

func (s *stubBoostLeads) FindHashIDByContactValue(ctx context.Context, value string) (string, error) {
	if value == "ops@affnet.com" {
		return s.hash, nil
	}
	return "", nil
}

func TestClassifyBoostPromoted(t *testing.T) {
	t.Parallel()
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc := &Service{registry: reg}
	status, hash, _ := svc.classifyBoost(context.Background(), sink.CrmBoostDoc{
		Snippet: "need voluum alternative for our team",
	})
	if status != sink.CrmBoostPromoted {
		t.Fatalf("status=%q want promoted hash=%q", status, hash)
	}
}

func TestClassifyBoostMerged(t *testing.T) {
	t.Parallel()
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc := &Service{
		registry: reg,
		leads:    &stubBoostLeads{hash: "abc123"},
	}
	status, hash, _ := svc.classifyBoost(context.Background(), sink.CrmBoostDoc{
		Snippet:     "voluum alternative pricing",
		ContactHint: "ops@affnet.com",
	})
	if status != sink.CrmBoostMerged || hash != "abc123" {
		t.Fatalf("status=%q hash=%q", status, hash)
	}
}

func TestRunBoostWorkerResolvesPending(t *testing.T) {
	t.Parallel()
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	crm := &stubBoostCRM{
		pending: []sink.CrmBoostDoc{{
			JunkID:  "junk-1",
			Snippet: "need voluum alternative for media buying team",
		}},
	}
	svc := &Service{crm: crm, registry: reg}
	svc.runBoostWorker(context.Background())
	if crm.last.junkID != "junk-1" {
		t.Fatalf("junk_id=%q", crm.last.junkID)
	}
	if crm.last.status != sink.CrmBoostPromoted {
		t.Fatalf("status=%q", crm.last.status)
	}
}
