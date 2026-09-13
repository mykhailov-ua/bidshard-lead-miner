package httpclient

import (
	"net/http"
	"testing"
)

func TestLooksCloudflareBlockedCFRay(t *testing.T) {
	h := http.Header{"Cf-Ray": []string{"abc"}}
	if !LooksCloudflareBlocked(http.StatusForbidden, h, nil) {
		t.Fatal("expected CF block")
	}
}
