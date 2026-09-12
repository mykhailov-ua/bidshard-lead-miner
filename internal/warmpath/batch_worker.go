package warmpath

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/worker"
)

const (
	batchPhaseCore   = "core"
	batchPhaseEngage = "engage"
)

// BatchWorkerConfig controls async Gemini Batch API flush and poll loops.
type BatchWorkerConfig struct {
	FlushInterval    time.Duration
	PollInterval     time.Duration
	SpillPath        string
	EngageSpillPath  string
	StatePath        string
	EngageEnabled    bool
	EnrichEnabled    bool
	EngageBatchSize  int
}

type pendingBatchJob struct {
	Name      string             `json:"name"`
	Phase     string             `json:"phase"`
	CreatedAt time.Time          `json:"created_at"`
	Records   []SpillRecord      `json:"records,omitempty"`
	Engage    []EngageSpillRecord `json:"engage,omitempty"`
}

// BatchWorker spills warm-path leads to JSONL and submits Gemini Batch jobs.
type BatchWorker struct {
	client      *gemini.Client
	spill       *BatchSpill
	engageSpill *EngageSpill
	cfg         BatchWorkerConfig
	svc         *Service

	mu    sync.Mutex
	state []pendingBatchJob
}

func NewBatchWorker(client *gemini.Client, spill *BatchSpill, cfg BatchWorkerConfig, svc *Service) *BatchWorker {
	cfg.FlushInterval = worker.DurationOr(cfg.FlushInterval, 30*time.Minute)
	cfg.PollInterval = worker.DurationOr(cfg.PollInterval, 2*time.Minute)
	cfg.EngageBatchSize = worker.IntOr(cfg.EngageBatchSize, 4)
	if cfg.SpillPath == "" {
		cfg.SpillPath = "data/runtime/gemini_batch_spill.jsonl"
	}
	if cfg.EngageSpillPath == "" {
		cfg.EngageSpillPath = "data/runtime/gemini_batch_engage_spill.jsonl"
	}
	if cfg.StatePath == "" {
		cfg.StatePath = "data/runtime/gemini_batch_jobs.json"
	}
	w := &BatchWorker{
		client:      client,
		spill:       spill,
		engageSpill: NewEngageSpill(cfg.EngageSpillPath),
		cfg:         cfg,
		svc:         svc,
	}
	w.loadState()
	return w
}

func (w *BatchWorker) Run(ctx context.Context, wg *sync.WaitGroup) {
	if w == nil || w.client == nil || w.spill == nil || w.svc == nil {
		return
	}
	worker.Run(ctx, wg, w.loop)
}

func (w *BatchWorker) loop(ctx context.Context) {
	slog.Info("gemini batch worker started",
		"flush_interval", w.cfg.FlushInterval,
		"poll_interval", w.cfg.PollInterval,
		"spill", w.spill.Path(),
		"engage_spill", w.engageSpill.Path(),
		"engage", w.cfg.EngageEnabled,
	)
	flushTicker := time.NewTicker(w.cfg.FlushInterval)
	pollTicker := time.NewTicker(w.cfg.PollInterval)
	defer flushTicker.Stop()
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-flushTicker.C:
			w.flushCoreSpill(ctx)
			w.flushEngageSpill(ctx)
		case <-pollTicker.C:
			w.pollJobs(ctx)
		}
	}
}

func (w *BatchWorker) Spill(batch []Event, geoClassify bool) error {
	if w == nil || w.spill == nil || len(batch) == 0 {
		return nil
	}
	key, err := newBatchKey("warm")
	if err != nil {
		return err
	}
	return w.spill.Append(SpillRecord{
		Key:         key,
		GeoClassify: geoClassify,
		Events:      append([]Event(nil), batch...),
	})
}

func (w *BatchWorker) flushCoreSpill(ctx context.Context) {
	records, err := w.spill.Drain()
	if err != nil {
		slog.Warn("gemini batch spill drain failed", "error", err)
		return
	}
	if len(records) == 0 {
		return
	}
	lines, valid := w.buildCoreLines(records)
	if len(lines) == 0 {
		slog.Warn("gemini batch flush skipped: no valid lines", "records", len(records))
		return
	}
	displayName := fmt.Sprintf("lip-warm-core-%s", time.Now().UTC().Format("20060102T%H%M%SZ"))
	job, err := w.submitLines(ctx, lines, displayName)
	if err != nil {
		slog.Warn("gemini batch core submit failed", "records", len(records), "error", err)
		if err := w.requeueCoreRecords(valid); err != nil {
			slog.Warn("gemini batch requeue failed", "error", err)
		}
		return
	}
	w.appendJob(pendingBatchJob{
		Name:      job.Name,
		Phase:     batchPhaseCore,
		CreatedAt: time.Now().UTC(),
		Records:   valid,
	})
	slog.Info("gemini batch core job submitted",
		"name", job.Name,
		"state", job.State,
		"requests", len(valid),
	)
}

func (w *BatchWorker) flushEngageSpill(ctx context.Context) {
	if !w.cfg.EngageEnabled {
		return
	}
	records, err := w.engageSpill.Drain()
	if err != nil {
		slog.Warn("gemini batch engage spill drain failed", "error", err)
		return
	}
	if len(records) == 0 {
		return
	}
	lines, valid := w.buildEngageLines(records)
	if len(lines) == 0 {
		return
	}
	displayName := fmt.Sprintf("lip-warm-engage-%s", time.Now().UTC().Format("20060102T%H%M%SZ"))
	job, err := w.submitLines(ctx, lines, displayName)
	if err != nil {
		slog.Warn("gemini batch engage submit failed", "records", len(records), "error", err)
		if err := w.requeueEngageRecords(valid); err != nil {
			slog.Warn("gemini batch engage requeue failed", "error", err)
		}
		return
	}
	w.appendJob(pendingBatchJob{
		Name:      job.Name,
		Phase:     batchPhaseEngage,
		CreatedAt: time.Now().UTC(),
		Engage:    valid,
	})
	slog.Info("gemini batch engage job submitted",
		"name", job.Name,
		"state", job.State,
		"requests", len(valid),
	)
}

func (w *BatchWorker) buildCoreLines(records []SpillRecord) ([]gemini.BatchJSONLLine, []SpillRecord) {
	lines := make([]gemini.BatchJSONLLine, 0, len(records))
	valid := make([]SpillRecord, 0, len(records))
	for _, rec := range records {
		inputs := eventsToLeadInputs(rec.Events, w.svc.cfg.GeoBlockCountries)
		line, err := gemini.BuildLeadBatchCoreJSONLLine(rec.Key, inputs, rec.GeoClassify)
		if err != nil {
			slog.Warn("gemini batch core line build failed", "key", rec.Key, "error", err)
			continue
		}
		lines = append(lines, gemini.BatchJSONLLine{Key: rec.Key, Request: trimJSONLRequest(line)})
		valid = append(valid, rec)
	}
	return lines, valid
}

func (w *BatchWorker) buildEngageLines(records []EngageSpillRecord) ([]gemini.BatchJSONLLine, []EngageSpillRecord) {
	lines := make([]gemini.BatchJSONLLine, 0, len(records))
	valid := make([]EngageSpillRecord, 0, len(records))
	for _, rec := range records {
		inputs := engageItemsToInputs(rec.Items, w.svc.cfg.GeoBlockCountries)
		line, err := gemini.BuildLeadBatchEngageJSONLLine(rec.Key, inputs, w.cfg.EnrichEnabled)
		if err != nil {
			slog.Warn("gemini batch engage line build failed", "key", rec.Key, "error", err)
			continue
		}
		lines = append(lines, gemini.BatchJSONLLine{Key: rec.Key, Request: trimJSONLRequest(line)})
		valid = append(valid, rec)
	}
	return lines, valid
}

func trimJSONLRequest(line []byte) json.RawMessage {
	var row gemini.BatchJSONLLine
	if err := json.Unmarshal(line, &row); err != nil {
		return json.RawMessage(line)
	}
	return row.Request
}

func (w *BatchWorker) submitLines(ctx context.Context, lines []gemini.BatchJSONLLine, displayName string) (*gemini.BatchJob, error) {
	return w.client.SubmitBatchJSONL(ctx, lines, displayName)
}

func (w *BatchWorker) pollJobs(ctx context.Context) {
	jobs := w.snapshotJobs()
	if len(jobs) == 0 {
		return
	}
	remaining := make([]pendingBatchJob, 0, len(jobs))
	for _, job := range jobs {
		status, err := w.client.GetBatchJob(ctx, job.Name)
		if err != nil {
			slog.Warn("gemini batch poll failed", "name", job.Name, "error", err)
			remaining = append(remaining, job)
			continue
		}
		switch status.State {
		case gemini.BatchStatePending, gemini.BatchStateRunning, "":
			remaining = append(remaining, job)
		case gemini.BatchStateSucceeded:
			var applyErr error
			if job.Phase == batchPhaseEngage {
				applyErr = w.applyEngageJobResults(ctx, job, status)
			} else {
				applyErr = w.applyCoreJobResults(ctx, job, status)
			}
			if applyErr != nil {
				slog.Warn("gemini batch apply failed", "name", job.Name, "phase", job.Phase, "error", applyErr)
				remaining = append(remaining, job)
			} else {
				slog.Info("gemini batch job completed", "name", job.Name, "phase", job.Phase)
			}
		case gemini.BatchStateFailed, gemini.BatchStateCancelled, gemini.BatchStateExpired:
			slog.Warn("gemini batch job terminal failure",
				"name", job.Name,
				"phase", job.Phase,
				"state", status.State,
				"error", status.Error,
			)
		default:
			remaining = append(remaining, job)
		}
	}
	w.replaceJobs(remaining)
}

func (w *BatchWorker) loadBatchOutput(ctx context.Context, status *gemini.BatchJob) (map[string][]byte, error) {
	byKey := make(map[string][]byte)
	if status.OutputFile != "" {
		raw, err := w.client.DownloadBatchFile(ctx, status.OutputFile)
		if err != nil {
			return nil, err
		}
		return gemini.ParseLeadBatchOutputJSONL(raw)
	}
	for _, row := range status.InlineResponses {
		if row.Error != "" {
			return nil, fmt.Errorf("inline key %s: %s", row.Key, row.Error)
		}
		byKey[row.Key] = row.Response
	}
	return byKey, nil
}

func (w *BatchWorker) applyCoreJobResults(ctx context.Context, job pendingBatchJob, status *gemini.BatchJob) error {
	byKey, err := w.loadBatchOutput(ctx, status)
	if err != nil {
		return err
	}
	recByKey := make(map[string]SpillRecord, len(job.Records))
	for _, rec := range job.Records {
		recByKey[rec.Key] = rec
	}
	highMin := highMinFromReg(w.svc.registry)
	var engageItems []EngageSpillItem
	for key, raw := range byKey {
		rec, ok := recByKey[key]
		if !ok {
			continue
		}
		inputs := eventsToLeadInputs(rec.Events, w.svc.cfg.GeoBlockCountries)
		results, err := gemini.ParseLeadBatchCoreResponse(raw, rec.GeoClassify, inputs)
		if err != nil {
			return fmt.Errorf("parse key %s: %w", key, err)
		}
		byID := make(map[string]Event, len(rec.Events))
		for _, ev := range rec.Events {
			byID[ev.HashID] = ev
		}
		for _, res := range results {
			ev, ok := byID[res.HashID]
			if !ok {
				continue
			}
			if w.shouldQueueEngage(ev, res, highMin) {
				engageItems = append(engageItems, EngageSpillItem{Event: ev, Core: res})
				w.svc.applyResultPending(ctx, ev, res, highMin)
				continue
			}
			w.svc.applyResult(ctx, ev, res, highMin)
		}
	}
	if len(engageItems) > 0 {
		if err := w.queueEngageItems(engageItems); err != nil {
			return err
		}
	}
	return nil
}

func (w *BatchWorker) applyEngageJobResults(ctx context.Context, job pendingBatchJob, status *gemini.BatchJob) error {
	byKey, err := w.loadBatchOutput(ctx, status)
	if err != nil {
		return err
	}
	recByKey := make(map[string]EngageSpillRecord, len(job.Engage))
	for _, rec := range job.Engage {
		recByKey[rec.Key] = rec
	}
	highMin := highMinFromReg(w.svc.registry)
	for key, raw := range byKey {
		rec, ok := recByKey[key]
		if !ok {
			continue
		}
		inputs := engageItemsToInputs(rec.Items, w.svc.cfg.GeoBlockCountries)
		core := make([]gemini.LeadBatchResult, 0, len(rec.Items))
		byID := make(map[string]Event, len(rec.Items))
		for _, item := range rec.Items {
			core = append(core, item.Core)
			byID[item.Event.HashID] = item.Event
		}
		results, err := gemini.ParseLeadBatchEngageResponse(raw, core, inputs, w.cfg.EnrichEnabled)
		if err != nil {
			return fmt.Errorf("parse engage key %s: %w", key, err)
		}
		for _, res := range results {
			ev, ok := byID[res.HashID]
			if !ok {
				continue
			}
			w.svc.applyResult(ctx, ev, res, highMin)
		}
	}
	return nil
}

func (w *BatchWorker) shouldQueueEngage(ev Event, res gemini.LeadBatchResult, highMin int) bool {
	if !w.cfg.EngageEnabled {
		return false
	}
	if w.svc.cfg.GeoClassifyEnabled && res.Geo.ShouldReject(w.svc.cfg.GeoBlockCountries) {
		return false
	}
	priority := w.svc.computeResultPriority(ev, res, highMin)
	if res.ICP.ICP == "none" && !res.ICP.Hot && priority == scoring.PriorityLow {
		return false
	}
	return scoring.MeetsMinPriority(priority, scoring.PriorityHigh)
}

func (w *BatchWorker) queueEngageItems(items []EngageSpillItem) error {
	chunk := w.cfg.EngageBatchSize
	if chunk <= 0 {
		chunk = 4
	}
	for i := 0; i < len(items); i += chunk {
		end := i + chunk
		if end > len(items) {
			end = len(items)
		}
		key, err := newBatchKey("engage")
		if err != nil {
			return err
		}
		rec := EngageSpillRecord{
			Key:   key,
			Items: append([]EngageSpillItem(nil), items[i:end]...),
		}
		if err := w.engageSpill.Append(rec); err != nil {
			return err
		}
	}
	return nil
}

func engageItemsToInputs(items []EngageSpillItem, blocked []string) []gemini.LeadBatchInput {
	events := make([]Event, 0, len(items))
	for _, item := range items {
		ev := item.Event
		ev.Priority = string(scoring.PriorityHigh)
		events = append(events, ev)
	}
	return eventsToLeadInputs(events, blocked)
}

func (w *BatchWorker) requeueCoreRecords(records []SpillRecord) error {
	for _, rec := range records {
		if err := w.spill.Append(rec); err != nil {
			return err
		}
	}
	return nil
}

func (w *BatchWorker) requeueEngageRecords(records []EngageSpillRecord) error {
	for _, rec := range records {
		if err := w.engageSpill.Append(rec); err != nil {
			return err
		}
	}
	return nil
}

func eventsToLeadInputs(events []Event, blocked []string) []gemini.LeadBatchInput {
	out := make([]gemini.LeadBatchInput, 0, len(events))
	for _, ev := range events {
		out = append(out, gemini.LeadBatchInput{
			ID:               ev.HashID,
			Source:           ev.Source,
			Priority:         ev.Priority,
			Score:            ev.Score,
			Snippet:          ev.Snippet,
			Contacts:         append([]string(nil), ev.Contacts...),
			ContactTypes:     append([]string(nil), ev.ContactTypes...),
			Stack:            append([]string(nil), ev.Stack...),
			Domain:           ev.Domain,
			RDAPCountry:      ev.RDAPCountry,
			DomainAgeDays:    ev.DomainAgeDays,
			DisplayName:      ev.DisplayName,
			BlockedCountries: append([]string(nil), blocked...),
		})
	}
	return out
}

func (w *BatchWorker) appendJob(job pendingBatchJob) {
	w.mu.Lock()
	w.state = append(w.state, job)
	w.mu.Unlock()
	w.saveState()
}

func (w *BatchWorker) snapshotJobs() []pendingBatchJob {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]pendingBatchJob(nil), w.state...)
}

func (w *BatchWorker) replaceJobs(jobs []pendingBatchJob) {
	w.mu.Lock()
	w.state = jobs
	w.mu.Unlock()
	w.saveState()
}

func (w *BatchWorker) loadState() {
	if w == nil || w.cfg.StatePath == "" {
		return
	}
	raw, err := os.ReadFile(w.cfg.StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		slog.Warn("gemini batch state load failed", "path", w.cfg.StatePath, "error", err)
		return
	}
	var jobs []pendingBatchJob
	if err := json.Unmarshal(raw, &jobs); err != nil {
		slog.Warn("gemini batch state decode failed", "path", w.cfg.StatePath, "error", err)
		return
	}
	w.state = jobs
}

func (w *BatchWorker) saveState() {
	if w == nil || w.cfg.StatePath == "" {
		return
	}
	w.mu.Lock()
	data, err := json.MarshalIndent(w.state, "", "  ")
	w.mu.Unlock()
	if err != nil {
		slog.Warn("gemini batch state encode failed", "error", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(w.cfg.StatePath), 0o755); err != nil {
		slog.Warn("gemini batch state mkdir failed", "error", err)
		return
	}
	if err := os.WriteFile(w.cfg.StatePath, data, 0o644); err != nil {
		slog.Warn("gemini batch state write failed", "error", err)
	}
}

func newBatchKey(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(prefix), hex.EncodeToString(b[:])), nil
}
