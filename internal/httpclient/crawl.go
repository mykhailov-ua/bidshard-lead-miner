package httpclient

import (
	"net/http"
	"time"
)

// CrawlClient returns a proxy-aware client when PARSER_PROXY_LIST is set, else Shared().
func CrawlClient(timeout time.Duration, proxyURLs []string) *http.Client {
	client, err := NewClientWithProxies(timeout, proxyURLs)
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
