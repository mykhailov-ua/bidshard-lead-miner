package httpclient

import (
	"net/http"
	"testing"
	"time"
)

func TestLastProxyIndexAfterRoundTrip(t *testing.T) {
	trans, err := newRotatingProxyTransport([]string{
		"http://10.0.0.1:8080",
		"http://10.0.0.2:8080",
	}, nil, "test", ProxyPoolConfig{PerProxyRPS: 100, PerProxyBurst: 8})
	if err != nil {
		t.Fatal(err)
	}
	trans.pool.endpoints[0].cooldown = time.Now().Add(time.Hour)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	// Dial via proxy may fail; index is set when a proxy is acquired.
	_, _ = trans.RoundTrip(req)
	if idx := trans.LastProxyIndex(); idx != 1 {
		t.Fatalf("LastProxyIndex=%d want 1", idx)
	}
	if idx := LastProxyIndex(&http.Client{Transport: trans}); idx != 1 {
		t.Fatalf("LastProxyIndex(client)=%d want 1", idx)
	}
}
