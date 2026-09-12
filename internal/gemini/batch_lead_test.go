package gemini

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildLeadBatchCoreJSONLLine_hasSchema(t *testing.T) {
	line, err := BuildLeadBatchCoreJSONLLine("warm-abc", []LeadBatchInput{{
		ID: "h1", Source: "tgweb", Snippet: "need voluum alternative",
	}}, true)
	if err != nil {
		t.Fatal(err)
	}
	var row batchJSONLLine
	if err := json.Unmarshal(line, &row); err != nil {
		t.Fatal(err)
	}
	if row.Key != "warm-abc" {
		t.Fatalf("key=%q", row.Key)
	}
	body := string(row.Request)
	if !strings.Contains(body, "responseSchema") {
		t.Fatalf("missing responseSchema: %s", body)
	}
	if !strings.Contains(body, "Analyze each lead") {
		t.Fatalf("missing prompt: %s", body)
	}
}

func TestParseLeadBatchCoreResponse(t *testing.T) {
	raw := []byte(`{"items":[{"id":"h1","blocked":false,"geo_confidence":"low","icp":"pro","hot":true,"spend_tier":"unknown","pilot_qualified":false}]}`)
	items := []LeadBatchInput{{ID: "h1", Source: "tgweb", ContactTypes: []string{"email"}}}
	out, err := ParseLeadBatchCoreResponse(raw, true, items)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].HashID != "h1" || out[0].ICP.ICP != "pro" || !out[0].ICP.Hot {
		t.Fatalf("unexpected %+v", out[0])
	}
}

func TestBuildLeadBatchEngageJSONLLine_hasSchema(t *testing.T) {
	line, err := BuildLeadBatchEngageJSONLLine("engage-1", []LeadBatchInput{{
		ID: "h1", Source: "tgweb", Priority: "High", Snippet: "need tracker",
		ContactTypes: []string{"telegram"},
	}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(line), "outreach_draft") {
		t.Fatalf("missing engage schema: %s", line)
	}
}

func TestParseLeadBatchEngageResponse(t *testing.T) {
	core := []LeadBatchResult{{HashID: "h1", ICP: ICPResult{ICP: "pro", Hot: true}}}
	items := []LeadBatchInput{{ID: "h1", Source: "tgweb", ContactTypes: []string{"telegram"}}}
	raw := []byte(`{"items":[{"id":"h1","outreach_channel":"telegram","outreach_angle":"tracker pain","outreach_draft":"Hi there","company_type":"media_buyer","enrich_summary":"buyer"}]}`)
	out, err := ParseLeadBatchEngageResponse(raw, core, items, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Engagement.OutreachDraft == "" || out[0].ICP.ICP != "pro" {
		t.Fatalf("unexpected %+v", out[0])
	}
}

func TestParseLeadBatchOutputJSONL(t *testing.T) {
	data := []byte(`{"key":"warm-1","response":{"candidates":[{"content":{"parts":[{"text":"{\"items\":[{\"id\":\"h1\",\"icp\":\"none\",\"hot\":false,\"spend_tier\":\"unknown\",\"pilot_qualified\":false}]}"}]}}]}}`)
	byKey, err := ParseLeadBatchOutputJSONL(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(byKey["warm-1"]) == 0 {
		t.Fatal("missing warm-1 response")
	}
}
