package sink

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// LeadDocReader loads a lead document by hash_id.
type LeadDocReader interface {
	GetLeadByHashID(ctx context.Context, hashID string) (LeadDoc, bool, error)
}

func (s *MongoStore) GetLeadByHashID(ctx context.Context, hashID string) (LeadDoc, bool, error) {
	if s == nil || hashID == "" {
		return LeadDoc{}, false, nil
	}
	var doc LeadDoc
	err := s.coll.FindOne(ctx, bson.M{"hash_id": hashID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return LeadDoc{}, false, nil
	}
	if err != nil {
		return LeadDoc{}, false, err
	}
	return doc, true, nil
}

// LeadDocExporter appends canonical lead export lines after warm-path patches.
type LeadDocExporter interface {
	AppendLeadDoc(ctx context.Context, doc LeadDoc) error
}

// ExportSyncPatcher patches Mongo then replays the full lead to JSON export.
type ExportSyncPatcher struct {
	inner   LeadAnalysisPatcher
	reader  LeadDocReader
	exporter LeadDocExporter
}

func NewExportSyncPatcher(inner LeadAnalysisPatcher, reader LeadDocReader, exporter LeadDocExporter) LeadAnalysisPatcher {
	if inner == nil || reader == nil || exporter == nil {
		return inner
	}
	return &ExportSyncPatcher{inner: inner, reader: reader, exporter: exporter}
}

func (p *ExportSyncPatcher) PatchLeadAnalysis(ctx context.Context, patch LeadAnalysisPatch) error {
	if p == nil || p.inner == nil {
		return nil
	}
	if err := p.inner.PatchLeadAnalysis(ctx, patch); err != nil {
		return err
	}
	if patch.HashID == "" || p.reader == nil || p.exporter == nil {
		return nil
	}
	doc, ok, err := p.reader.GetLeadByHashID(ctx, patch.HashID)
	if err != nil || !ok {
		return err
	}
	return p.exporter.AppendLeadDoc(ctx, doc)
}
