package httpclient

import (
	"net/http"
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

func httpclientReset(t *testing.T) {
	t.Helper()
	ResetForTest()
}
