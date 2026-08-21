package filter

import "testing"

func TestRejectHTMLBoilerplate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		text   string
		reject bool
	}{
		{
			name:   "meta junk",
			text:   "UTF-8 viewport width=device-width, initial-scale=1.0 X-UA-Compatible IE=edge",
			reject: true,
		},
		{
			name:   "real pain",
			text:   "voluum alternative needed. postback failing on FTD again. contact partnerships@example.com",
			reject: false,
		},
		{
			name:   "markup noise",
			text:   "><script nonce= type= data-cookieconsent= ignore Cookie async></script><style>",
			reject: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reject, _ := RejectHTMLBoilerplate(tc.text)
			if reject != tc.reject {
				t.Fatalf("reject=%v want %v text=%q", reject, tc.reject, tc.text)
			}
		})
	}
}
