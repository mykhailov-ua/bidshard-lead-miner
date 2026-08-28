package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/metrics"
)

// ProxyBudget gates proxy egress when daily cap is exceeded.
type ProxyBudget interface {
	Allow() bool
	Record(n int64)
}

var budgetGovernor ProxyBudget

// ErrProxyBudgetExceeded is returned when PARSER_PROXY_DAILY_MB_CAP is reached.
var ErrProxyBudgetExceeded = fmt.Errorf("proxy daily budget exceeded")

// SetProxyBudget wires the process-wide proxy egress governor (nil disables gating).
func SetProxyBudget(g ProxyBudget) {
	budgetGovernor = g
}

var (
	sharedOnce sync.Once
	shared     *http.Client
)

const defaultSharedTimeout = 30 * time.Second

// RotatingProxyTransport implements http.RoundTripper with lock-free proxy selection,
// per-proxy rate limits, uTLS TLS fingerprinting, browser header mirroring, and block cooldowns.
type RotatingProxyTransport struct {
	pool         *proxyPool
	baseTrans    http.RoundTripper
	transport    *http.Transport
	currentProxy atomic.Pointer[url.URL]
	sourceID     string
}

func NewRotatingProxyTransport(proxyURLs []string, baseTrans http.RoundTripper) (*RotatingProxyTransport, error) {
	return newRotatingProxyTransport(proxyURLs, baseTrans, "")
}

func newRotatingProxyTransport(proxyURLs []string, baseTrans http.RoundTripper, sourceID string, poolCfg ...ProxyPoolConfig) (*RotatingProxyTransport, error) {
	if err := config.ValidateProxyURLs(proxyURLs); err != nil {
		return nil, err
	}
	if baseTrans == nil {
		tTrans := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20, // tgweb multi-domain crawl; idle expiry -> connect churn in BPF, not always a leak
			IdleConnTimeout:     90 * time.Second,
			ForceAttemptHTTP2:   true,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Terminate TLS with Chrome ClientHello; skip verify because residential proxies often MITM HTTPS.
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
					_ = rawConn.Close()
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
	t := &RotatingProxyTransport{
		pool:      newProxyPool(parsed, activePoolConfig(poolCfg...), sourceID),
		baseTrans: baseTrans,
		sourceID:  strings.TrimSpace(sourceID),
	}
	if tTrans, ok := baseTrans.(*http.Transport); ok {
		cloned := tTrans.Clone()
		cloned.Proxy = t.proxyFunc
		t.transport = cloned
	}
	return t, nil
}

func (t *RotatingProxyTransport) proxyFunc(req *http.Request) (*url.URL, error) {
	if t.pool == nil || len(t.pool.endpoints) == 0 {
		return nil, nil
	}
	proxy := t.currentProxy.Load()
	if proxy == nil {
		return nil, nil
	}
	return proxy, nil
}

func (t *RotatingProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(t.pool.endpoints) > 0 && budgetGovernor != nil && !budgetGovernor.Allow() {
		metrics.RecordProxyBudgetSkipped(t.sourceID)
		return nil, ErrProxyBudgetExceeded
	}

	proxy, err := t.pool.acquire(req.Context())
	if err != nil {
		return nil, err
	}
	t.currentProxy.Store(proxy)

	cloned := req.Clone(req.Context())
	applyBrowserHeaders(cloned)

	if t.transport != nil {
		resp, err := t.transport.RoundTrip(cloned)
		if err != nil && t.sourceID != "" {
			metrics.RecordProxyTransportFail(t.sourceID)
		}
		t.checkProxyBlockCooldown(proxy, resp)
		t.recordEgress(resp)
		return resp, err
	}

	resp, err := t.baseTrans.RoundTrip(cloned)
	if err != nil && t.sourceID != "" {
		metrics.RecordProxyTransportFail(t.sourceID)
	}
	t.checkProxyBlockCooldown(proxy, resp)
	t.recordEgress(resp)
	return resp, err
}

func (t *RotatingProxyTransport) recordEgress(resp *http.Response) {
	if resp == nil || t.sourceID == "" {
		return
	}
	if resp.ContentLength > 0 {
		if budgetGovernor != nil {
			budgetGovernor.Record(resp.ContentLength)
		}
		metrics.RecordProxyEgressBytes(t.sourceID, resp.ContentLength)
	}
}

func (t *RotatingProxyTransport) checkProxyBlockCooldown(proxy *url.URL, resp *http.Response) {
	if resp == nil || proxy == nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		DiscardResponseBody(resp, 64<<10)
		t.pool.markCooldown(proxy, "http_429")
		return
	case http.StatusForbidden, http.StatusServiceUnavailable:
		if resp.Header.Get("CF-Ray") != "" || strings.Contains(strings.ToLower(resp.Header.Get("Server")), "cloudflare") {
			DiscardResponseBody(resp, 64<<10)
			t.pool.markCooldown(proxy, fmt.Sprintf("http_%d", resp.StatusCode))
			return
		}
		prefix := peekBodyPrefix(resp, 8<<10)
		if isProxyBlock(resp, prefix) {
			DiscardResponseBody(resp, 64<<10)
			t.pool.markCooldown(proxy, fmt.Sprintf("http_%d", resp.StatusCode))
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
	// Default direct-egress client (supply seed fetch, enrich RDAP/DNS, etc.).
	// HTTP crawlers that honor PARSER_PROXY_LIST build a separate client via NewClientWithProxies.
	// Timeout is fixed on first call (deps warm-up); later calls ignore timeout to avoid races.
	sharedOnce.Do(func() {
		if timeout <= 0 {
			timeout = defaultSharedTimeout
		}
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
	return shared
}

// ClientWithSharedTransport reuses Shared()'s connection pool with a separate request timeout.
// Gemini uses a longer Timeout than crawl; sharing Transport avoids a second idle conn pool.
func ClientWithSharedTransport(requestTimeout time.Duration) *http.Client {
	if requestTimeout <= 0 {
		requestTimeout = defaultSharedTimeout
	}
	return &http.Client{
		Transport: Shared(requestTimeout).Transport,
		Timeout:   requestTimeout,
	}
}

func NewClientWithProxies(timeout time.Duration, proxies []string, sourceID string) (*http.Client, error) {
	trans, err := newRotatingProxyTransport(proxies, nil, sourceID)
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
