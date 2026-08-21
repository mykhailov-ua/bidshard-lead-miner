package sources

import "testing"

func TestParseSourceList(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"":             nil,
		"all":          {"forum", "supply", "lander", "reddit", "discord", "warrior", "serp"},
		"forum,reddit": {"forum", "reddit"},
	}
	for in, want := range cases {
		got := parseSourceList(in)
		if len(got) != len(want) {
			t.Fatalf("input=%q got=%v want=%v", in, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("input=%q got=%v want=%v", in, got, want)
			}
		}
	}
}
