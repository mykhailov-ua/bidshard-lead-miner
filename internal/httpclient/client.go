package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

var (
	sharedOnce sync.Once
	shared     *http.Client
)

// RotatingProxyTransport implements http.RoundTripper with lock-free proxy selection,
// uTLS TLS fingerprinting (Chrome_Auto), browser header mirroring, and 10-min Cloudflare cooldowns.
type RotatingProxyTransport struct {
	counter   uint64
	proxies   []*url.URL
	baseTrans http.RoundTripper
	cooldowns map[string]time.Time
	coolMu    sync.Mutex
}

func NewRotatingProxyTransport(proxyURLs []string, baseTrans http.RoundTripper) (*RotatingProxyTransport, error) {
	if baseTrans == nil {
		tTrans := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			ForceAttemptHTTP2:   true,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}
				rawConn, err := dialer.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				host, _, _ := net.SplitHostPort(addr)
				uConn := utls.UClient(rawConn, &utls.Config{
					ServerName:         host,
					InsecureSkipVerify: true,
					NextProtos:         []string{"h2", "http/1.1"},
				}, utls.HelloChrome_Auto)
				if err := uConn.HandshakeContext(ctx); err != nil {
					rawConn.Close()
					return nil, err
				}
				return uConn, nil
			},
		}
		_ = http2.ConfigureTransport(tTrans)
		baseTrans = tTrans
	}
	var parsed []*url.URL
	for _, raw := range proxyURLs {
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", raw, err)
		}
		parsed = append(parsed, u)
	}
	return &RotatingProxyTransport{
		proxies:   parsed,
		baseTrans: baseTrans,
		cooldowns: make(map[string]time.Time),
	}, nil
}

func (t *RotatingProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	applyBrowserHeaders(cloned)

	if len(t.proxies) == 0 {
		return t.baseTrans.RoundTrip(cloned)
	}

	proxy := t.selectActiveProxy()
	if proxy != nil {
		if tTrans, ok := t.baseTrans.(*http.Transport); ok {
			clonedTrans := tTrans.Clone()
			clonedTrans.Proxy = http.ProxyURL(proxy)
			resp, err := clonedTrans.RoundTrip(cloned)
			t.checkCloudflareCooldown(proxy, resp)
			return resp, err
		}
	}

	return t.baseTrans.RoundTrip(cloned)
}

func (t *RotatingProxyTransport) selectActiveProxy() *url.URL {
	t.coolMu.Lock()
	now := time.Now()
	var available []*url.URL
	for _, p := range t.proxies {
		if until, ok := t.cooldowns[p.String()]; !ok || now.After(until) {
			available = append(available, p)
		}
	}
	t.coolMu.Unlock()

	if len(available) == 0 {
		idx := atomic.AddUint64(&t.counter, 1) % uint64(len(t.proxies))
		return t.proxies[idx]
	}

	idx := atomic.AddUint64(&t.counter, 1) % uint64(len(available))
	return available[idx]
}

func (t *RotatingProxyTransport) checkCloudflareCooldown(proxy *url.URL, resp *http.Response) {
	if resp == nil || proxy == nil {
		return
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		isCF := resp.Header.Get("CF-Ray") != "" || strings.Contains(strings.ToLower(resp.Header.Get("Server")), "cloudflare")
		if isCF {
			t.coolMu.Lock()
			t.cooldowns[proxy.String()] = time.Now().Add(10 * time.Minute)
			t.coolMu.Unlock()
			slog.Warn("cloudflare block detected, proxy in 10m cooldown", "proxy", proxy.Redacted(), "status", resp.StatusCode)
		}
	}
}

func applyBrowserHeaders(req *http.Request) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	}
	if req.Header.Get("Sec-Ch-Ua") == "" {
		req.Header.Set("Sec-Ch-Ua", `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`)
	}
	if req.Header.Get("Sec-Ch-Ua-Mobile") == "" {
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	}
	if req.Header.Get("Sec-Ch-Ua-Platform") == "" {
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	}
	if req.Header.Get("Sec-Fetch-Dest") == "" {
		req.Header.Set("Sec-Fetch-Dest", "document")
	}
	if req.Header.Get("Sec-Fetch-Mode") == "" {
		req.Header.Set("Sec-Fetch-Mode", "navigate")
	}
	if req.Header.Get("Sec-Fetch-Site") == "" {
		req.Header.Set("Sec-Fetch-Site", "none")
	}
	if req.Header.Get("Sec-Fetch-User") == "" {
		req.Header.Set("Sec-Fetch-User", "?1")
	}
}

func Shared(timeout time.Duration) *http.Client {
	sharedOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
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

func NewClientWithProxies(timeout time.Duration, proxies []string) (*http.Client, error) {
	trans, err := NewRotatingProxyTransport(proxies, nil)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: trans,
		Timeout:   timeout,
	}, nil
}

func ResetForTest() {
	sharedOnce = sync.Once{}
	shared = nil
}


