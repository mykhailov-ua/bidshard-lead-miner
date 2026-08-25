package httpclient

import (
	"net/http"
	"time"
)

// CrawlClient returns a proxy-aware client when proxyURLs is non-empty, else Shared().
func CrawlClient(timeout time.Duration, proxyURLs []string, sourceID string) *http.Client {
	client, err := NewClientWithProxies(timeout, proxyURLs, sourceID)
	if err != nil {
		return Shared(timeout)
	}
	return client
}

// DoBytes runs req, reads up to limit bytes, and always closes the response body.
func DoBytes(client *http.Client, req *http.Request, limit int64) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	body, err := ReadResponseBody(resp, limit)
	return body, resp.StatusCode, err
}
