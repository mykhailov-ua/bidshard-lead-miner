package gemini

import "testing"

func TestGeoResultShouldReject(t *testing.T) {
	blocked := []string{"RU", "BY"}

	cases := []struct {
		name   string
		result GeoResult
		want   bool
	}{
		{
			name:   "high blocked flag",
			result: GeoResult{Blocked: true, Confidence: "high"},
			want:   true,
		},
		{
			name:   "low confidence ignored",
			result: GeoResult{Blocked: true, Confidence: "low"},
			want:   false,
		},
		{
			name:   "company country BY",
			result: GeoResult{CompanyCountry: "BY", Confidence: "medium"},
			want:   true,
		},
		{
			name:   "unknown countries pass",
			result: GeoResult{Blocked: false, Confidence: "medium", PersonCountry: "unknown"},
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.ShouldReject(blocked); got != tc.want {
				t.Fatalf("ShouldReject()=%v want %v result=%+v", got, tc.want, tc.result)
			}
		})
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	if got := normalizeCountryCode("ru"); got != "RU" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeCountryCode("unknown"); got != "unknown" {
		t.Fatalf("got %q", got)
	}
}
