package httpclient

import "net/http"

// LastProxyIndex returns the proxy pool index used for the last RoundTrip on this
// client, or -1 when the transport is not a rotating proxy pool or no request ran yet.
func LastProxyIndex(client *http.Client) int {
	if client == nil || client.Transport == nil {
		return -1
	}
	t, ok := client.Transport.(*RotatingProxyTransport)
	if !ok {
		return -1
	}
	return t.LastProxyIndex()
}
