package gemini

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildLeadBatchCoreJSONLLine builds one Gemini Batch API JSONL row for a multi-lead core pass.
func BuildLeadBatchCoreJSONLLine(key string, items []LeadBatchInput, geoClassify bool) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("gemini batch lead: empty key")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("gemini batch lead: empty items")
	}
	payload, err := marshalLeadBatchPayload(items)
	if err != nil {
		return nil, err
	}
	sysPrompt := leadBatchCoreSystemPrompt
	if !geoClassify {
		sysPrompt = leadBatchCoreICPSystemPrompt
	}
	prompt := "Analyze each lead. Return one output item per input id.\n\n" + string(payload)
	req := generateRequest{
		Contents:          []content{{Parts: []part{{Text: prompt}}}},
		SystemInstruction: &content{Parts: []part{{Text: sysPrompt}}},
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   leadBatchCoreSchema(geoClassify),
		},
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	line := batchJSONLLine{Key: key, Request: reqBytes}
	return json.Marshal(line)
}

// ParseLeadBatchCoreResponse decodes one batch output row into LeadBatchResult values.
func ParseLeadBatchCoreResponse(raw []byte, geoClassify bool, items []LeadBatchInput) ([]LeadBatchResult, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("gemini batch lead: empty response")
	}
	var parsed leadBatchResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return nil, err
	}
	byID := indexLeadBatchInputs(items)
	out := make([]LeadBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, leadBatchResultFromItem(item, geoClassify, byID[strings.TrimSpace(item.ID)]))
	}
	return out, nil
}

// BuildLeadBatchEngageJSONLLine builds one Batch API row for a High-only engage/enrich pass.
func BuildLeadBatchEngageJSONLLine(key string, items []LeadBatchInput, enrich bool) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("gemini batch engage: empty key")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("gemini batch engage: empty items")
	}
	payload, err := marshalLeadBatchPayload(items)
	if err != nil {
		return nil, err
	}
	prompt := "Draft outreach for each High-priority lead id.\n\n" + string(payload)
	req := generateRequest{
		Contents:          []content{{Parts: []part{{Text: prompt}}}},
		SystemInstruction: &content{Parts: []part{{Text: leadBatchEngageSystemPrompt}}},
		GenerationConfig: generationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   leadBatchEngageSchema,
		},
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	line := batchJSONLLine{Key: key, Request: reqBytes}
	return json.Marshal(line)
}

// ParseLeadBatchEngageResponse decodes engage batch output and merges onto core results.
func ParseLeadBatchEngageResponse(raw []byte, core []LeadBatchResult, items []LeadBatchInput, enrich bool) ([]LeadBatchResult, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("gemini batch engage: empty response")
	}
	var parsed leadBatchResponse
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return nil, err
	}
	byID := indexLeadBatchInputs(items)
	coreByID := make(map[string]LeadBatchResult, len(core))
	for _, r := range core {
		coreByID[r.HashID] = r
	}
	out := make([]LeadBatchResult, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		id := strings.TrimSpace(item.ID)
		base := coreByID[id]
		base.HashID = id
		in := byID[id]
		base.Engagement = EngagementResult{
			OutreachChannel: normalizeOutreachChannel(item.OutreachChannel),
			OutreachSubject: strings.TrimSpace(item.OutreachSubject),
			OutreachAngle:   strings.TrimSpace(item.OutreachAngle),
			OutreachDraft:   strings.TrimSpace(item.OutreachDraft),
		}
		base.Engagement = ReconcileEngagement(base.Engagement, in.ContactTypes, in.Source)
		if enrich {
			base.Enrichment = EnrichSynthResult{
				CompanyType:   normalizeCompanyType(item.CompanyType),
				GeoConfidence: normalizeConfidence(item.EnrichGeoConfidence),
				Summary:       strings.TrimSpace(item.EnrichSummary),
			}
		}
		qualified, tags := ApplyEngagementPilot(base.Engagement)
		base.PilotQualified = qualified
		base.PilotTags = tags
		out = append(out, base)
	}
	return out, nil
}

// ParseLeadBatchOutputJSONL parses a downloaded batch result file.
func ParseLeadBatchOutputJSONL(data []byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Key      string            `json:"key"`
			Response *generateResponse `json:"response"`
			Error    *apiError         `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("gemini batch output line: %w", err)
		}
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		if row.Error != nil {
			return nil, fmt.Errorf("gemini batch key %s: %s", key, row.Error.Message)
		}
		text := batchResponseText(row.Response)
		if len(text) == 0 {
			return nil, fmt.Errorf("gemini batch key %s: empty response", key)
		}
		out[key] = text
	}
	return out, nil
}
