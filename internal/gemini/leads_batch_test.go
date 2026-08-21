package gemini

import "testing"

func TestLeadBatchResultFromItemSkipsGeoWhenDisabled(t *testing.T) {
	t.Parallel()

	item := leadBatchItem{
		ID:            "x1",
		Blocked:       true,
		GeoConfidence: "high",
		PersonCountry: "RU",
		ICP:           "pro",
		Hot:           true,
		SpendTier:     "15k-150k",
	}
	res := leadBatchResultFromItem(item, false)
	if res.Geo.Blocked {
		t.Fatal("expected empty geo when geoClassify=false")
	}
	if res.ICP.ICP != "pro" {
		t.Fatalf("icp=%q", res.ICP.ICP)
	}
}

func TestLeadBatchResultFromItemGeoWhenEnabled(t *testing.T) {
	t.Parallel()

	item := leadBatchItem{
		ID:            "x2",
		Blocked:       true,
		GeoConfidence: "high",
		PersonCountry: "RU",
	}
	res := leadBatchResultFromItem(item, true)
	if !res.Geo.Blocked {
		t.Fatal("expected geo blocked when geoClassify=true")
	}
}
