package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

// ReadResponseBody reads up to limit bytes and always closes resp.Body.
func ReadResponseBody(resp *http.Response, limit int64) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

// DiscardResponseBody drains up to limit bytes so the transport can reuse the connection.
// Leaves an empty body so callers can still invoke ReadResponseBody safely.
func DiscardResponseBody(resp *http.Response, limit int64) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, limit))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(nil))
}
