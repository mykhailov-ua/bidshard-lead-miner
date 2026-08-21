package webhook

import "time"

// Keep inbound POST bounded so a slow parser cannot tie up the listener.
const (
	readHeaderTimeout = 2 * time.Second
	readTimeout       = 5 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 30 * time.Second
)
