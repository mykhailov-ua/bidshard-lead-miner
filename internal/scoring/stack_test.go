package scoring

import "testing"

func TestCollectStackFromNextData(t *testing.T) {
	t.Parallel()
	html := `<html><script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"tracker":"voluumtrk.example"}}}</script></html>`
	stack, structured := CollectStack(html)
	if len(stack) == 0 {
		t.Fatal("expected stack from __NEXT_DATA__")
	}
	if !structured {
		t.Fatal("expected structured stack flag")
	}
}

func TestCollectStackMergesHTMLAndJSON(t *testing.T) {
	t.Parallel()
	html := `<script id="__NEXT_DATA__" type="application/json">{"vendor":"keitaro"}</script>`
	stack, _ := CollectStack(html)
	if len(stack) == 0 {
		t.Fatal("expected keitaro stack")
	}
}
