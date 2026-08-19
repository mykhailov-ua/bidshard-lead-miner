package httpclient

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestSharedClientIsSingleton(t *testing.T) {
	httpclientReset(t)

	c1 := Shared(2 * time.Second)
	c2 := Shared(3 * time.Second)
	if c1 != c2 {
		t.Fatal("expected singleton shared client")
	}
}

func TestSharedTransportSettings(t *testing.T) {
	httpclientReset(t)

	client := Shared(5 * time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type %T", client.Transport)
	}
	if transport.MaxIdleConns != 100 {
		t.Fatalf("MaxIdleConns=%d want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 20 {
		t.Fatalf("MaxIdleConnsPerHost=%d want 20", transport.MaxIdleConnsPerHost)
	}
	if transport.DialContext == nil {
		t.Fatal("expected DialContext")
	}
}

func TestRotatingProxyTransport(t *testing.T) {
	proxies := []string{
		"http://10.0.0.1:8080",
		"http://10.0.0.2:8080",
		"http://10.0.0.3:8080",
	}
	trans, err := NewRotatingProxyTransport(proxies, nil)
	if err != nil {
		t.Fatalf("failed to create proxy transport: %v", err)
	}
	if len(trans.proxies) != 3 {
		t.Fatalf("expected 3 parsed proxies, got %d", len(trans.proxies))
	}
}

func BenchmarkRotatingProxySelection(b *testing.B) {
	proxies := []string{
		"http://10.0.0.1:8080",
		"http://10.0.0.2:8080",
		"http://10.0.0.3:8080",
		"http://10.0.0.4:8080",
	}
	trans, _ := NewRotatingProxyTransport(proxies, nil)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := atomic.AddUint64(&trans.counter, 1) % uint64(len(trans.proxies))
		_ = trans.proxies[idx]
		_ = req
	}
}

func httpclientReset(t *testing.T) {
	t.Helper()
	ResetForTest()
}

