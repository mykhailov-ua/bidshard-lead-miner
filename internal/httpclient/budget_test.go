package httpclient

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

type stubBudget struct {
	allow bool
}

func (s *stubBudget) Allow() bool  { return s.allow }
func (s *stubBudget) Record(int64) {}

func TestRotatingProxyTransportBlocksWhenBudgetExceeded(t *testing.T) {
	SetProxyBudget(&stubBudget{allow: false})
	t.Cleanup(func() { SetProxyBudget(nil) })

	client, err := NewClientWithProxies(2*time.Second, []string{"http://127.0.0.1:9"}, "forum")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if !errors.Is(err, ErrProxyBudgetExceeded) {
		t.Fatalf("err=%v want %v", err, ErrProxyBudgetExceeded)
	}
}
