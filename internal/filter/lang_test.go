package filter

import "testing"

func TestRejectLongCyrillicWithoutLatin(t *testing.T) {
	t.Parallel()
	reject, _ := RejectLongCyrillicWithoutLatin(stringsRepeatCyrillic(50))
	if !reject {
		t.Fatal("expected reject")
	}
	ok, _ := RejectLongCyrillicWithoutLatin(stringsRepeatCyrillic(50) + " voluum alternative postback")
	if ok {
		t.Fatal("expected pass with EN pain hint")
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"Buscando alternativa a voluum porque es muy caro", "es"},
		{"Procurando alternativa ao keitaro muito caro", "pt"},
		{"Szukam alternatywy dla voluum tracker nie dziala", "pl"},
		{"Tracker zu teuer suche alternative", "de"},
		{"Tracker trop cher alternative a voluum", "fr"},
		{"Looking for voluum alternative", "en"},
		{stringsRepeatCyrillic(50), "ru"},
	}

	for _, tt := range tests {
		got := DetectLanguage(tt.text)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.text, got, tt.want)
		}
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
