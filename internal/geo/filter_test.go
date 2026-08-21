package geo

import "testing"

func TestFilterRejectCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		text    string
		contact string
		reason  string
	}{
		{"ru email domain", "contact buyer@team.ru please", "", "ru domain"},
		{"rf punycode domain", "write to sales@shop.рф", "", "ru domain"},
		{"by email domain", "ops@agency.by", "", "by domain"},
		{"bel punycode domain", "info@shop.бел", "", "by domain"},
		{"ru phone", "call +7 916 555 1212", "", "ru phone"},
		{"by phone", "whatsapp +375 29 123 4567", "", "by phone"},
		{"moscow bio", "based in Moscow, media buyer", "", "ru/by bio signal"},
		{"minsk bio", "team from Minsk", "", "ru/by bio signal"},
		{"russia bio", "traffic from Russia only", "", "ru/by bio signal"},
		{"belarus bio", "office in Belarus", "", "ru/by bio signal"},
		{"europe moscow tz", "timezone Europe/Moscow", "", "ru/by bio signal"},
		{"cyrillic only", "ищем альтернативу трекеру для арбитража без английского", "", "cyrillic-only context"},
		{"cyrillic heavy", stringsRepeatCyrillic(25) + " tracker voluum", "", "cyrillic-heavy context"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Filter(tc.text, tc.contact)
			if got.OK {
				t.Fatalf("expected reject, got pass")
			}
			if got.Reason != tc.reason {
				t.Fatalf("reason=%q want %q", got.Reason, tc.reason)
			}
		})
	}
}

func TestFilterPassCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		text    string
		contact string
	}{
		{"us email", "voluum alternative", "ops@igaming-team.com"},
		{"latam telegram", "postback failing on FTD", "telegram:@buyer_mx"},
		{"mixed en cyrillic", "voluum alternative для команды in LATAM", "buyer@mx-casino.com"},
		{"uk domain", "tracker migration", "ops@agency.co.uk"},
		{"hostname seed", "traffic-moscow.example.com", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Filter(tc.text, tc.contact)
			if !got.OK {
				t.Fatalf("expected pass, got reject: %s", got.Reason)
			}
		})
	}
}

func stringsRepeatCyrillic(n int) string {
	r := 'а'
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}

func TestIsBlockedTLD(t *testing.T) {
	t.Parallel()
	if !IsBlockedTLD("shop.example.ru") {
		t.Fatal("expected .ru blocked")
	}
	if !IsBlockedTLD("www.agency.by") {
		t.Fatal("expected .by blocked")
	}
	if IsBlockedTLD("bojoko.com") {
		t.Fatal("expected .com allowed")
	}
}
