package httpclient

import "net/http"

// LooksCloudflareBlocked reports common Cloudflare / anti-bot block responses.
func LooksCloudflareBlocked(status int, header http.Header, body []byte) bool {
	resp := &http.Response{StatusCode: status, Header: header}
	return isProxyBlock(resp, body)
}
