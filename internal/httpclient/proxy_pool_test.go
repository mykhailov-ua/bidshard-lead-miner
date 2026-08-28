package httpclient

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestPickEndpointSkipsCooledProxy(t *testing.T) {
	u1, _ := url.Parse("http://10.0.0.1:8080")
	u2, _ := url.Parse("http://10.0.0.2:8080")
	pool := newProxyPool([]*url.URL{u1, u2}, ProxyPoolConfig{
		PerProxyRPS:      10,
		PerProxyBurst:    4,
		CooldownDuration: time.Minute,
	}, "test")
	pool.endpoints[0].cooldown = time.Now().Add(time.Hour)

	ep, wait := pool.pickEndpoint()
	if ep == nil || wait != (time.Time{}) {
		t.Fatalf("expected available proxy, got ep=%v wait=%v", ep, wait)
	}
	if ep.url.String() != u2.String() {
		t.Fatalf("proxy=%s want %s", ep.url, u2)
	}
}

func TestPickEndpointWaitsWhenAllCooled(t *testing.T) {
	u1, _ := url.Parse("http://10.0.0.1:8080")
	pool := newProxyPool([]*url.URL{u1}, DefaultProxyPoolConfig(), "test")
	until := time.Now().Add(2 * time.Second)
	pool.endpoints[0].cooldown = until

	ep, wait := pool.pickEndpoint()
	if ep != nil {
		t.Fatalf("expected nil proxy, got %v", ep)
	}
	if wait.Before(until.Add(-time.Second)) || wait.After(until.Add(time.Second)) {
		t.Fatalf("wait=%v want ~%v", wait, until)
	}
}

func TestPerProxyRateLimit(t *testing.T) {
	u1, _ := url.Parse("http://10.0.0.1:8080")
	pool := newProxyPool([]*url.URL{u1}, ProxyPoolConfig{
		PerProxyRPS:      2,
		PerProxyBurst:    1,
		CooldownDuration: time.Minute,
	}, "test")

	ctx := context.Background()
	start := time.Now()
	if _, err := pool.acquire(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.acquire(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("expected rate limit wait, elapsed=%s", elapsed)
	}
}

func TestIsProxyBlockHeadersAndBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"Cf-Ray": []string{"abc"}},
	}
	if !isProxyBlock(resp, nil) {
		t.Fatal("expected CF-Ray block")
	}

	resp = &http.Response{StatusCode: http.StatusForbidden}
	if isProxyBlock(resp, []byte("ok page")) {
		t.Fatal("unexpected block on normal 403")
	}
	if !isProxyBlock(resp, []byte("<title>Access Denied</title>")) {
		t.Fatal("expected access denied block")
	}
}

func TestRotatingProxyTransportNoCooledFallback(t *testing.T) {
	proxies := []string{
		"http://10.0.0.1:8080",
		"http://10.0.0.2:8080",
	}
	trans, err := newRotatingProxyTransport(proxies, nil, "tgweb", ProxyPoolConfig{
		PerProxyRPS:      100,
		PerProxyBurst:    8,
		CooldownDuration: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(time.Hour)
	for _, ep := range trans.pool.endpoints {
		ep.cooldown = now
	}
	ep, wait := trans.pool.pickEndpoint()
	if ep != nil || wait.IsZero() {
		t.Fatalf("expected all cooled, ep=%v wait=%v", ep, wait)
	}
}

func TestProxyPoolRoundRobinAmongAvailable(t *testing.T) {
	u1, _ := url.Parse("http://10.0.0.1:8080")
	u2, _ := url.Parse("http://10.0.0.2:8080")
	pool := newProxyPool([]*url.URL{u1, u2}, ProxyPoolConfig{PerProxyRPS: 100, PerProxyBurst: 8}, "test")

	var seen int64
	for i := 0; i < 4; i++ {
		ep, _ := pool.pickEndpoint()
		if ep.url.Host == "10.0.0.2:8080" {
			atomic.AddInt64(&seen, 1)
		}
	}
	if seen == 0 {
		t.Fatal("expected rotation across proxies")
	}
}
