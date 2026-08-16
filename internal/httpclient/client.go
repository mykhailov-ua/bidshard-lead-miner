package httpclient

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	sharedOnce sync.Once
	shared     *http.Client
)

func Shared(timeout time.Duration) *http.Client {
	sharedOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
		shared = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	})
	if timeout > 0 {
		shared.Timeout = timeout
	}
	return shared
}

func ResetForTest() {
	sharedOnce = sync.Once{}
	shared = nil
}
