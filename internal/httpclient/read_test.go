package httpclient

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDiscardResponseBodyAllowsReadResponseBody(t *testing.T) {
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader("cloudflare block page")),
	}
	DiscardResponseBody(resp, 1024)
	body, err := ReadResponseBody(resp, 1024)
	if err != nil {
		t.Fatalf("ReadResponseBody after discard: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("body=%q want empty after discard", body)
	}
}
