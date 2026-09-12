package telethon

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// IPCServer listens on a Unix domain socket for Telethon sidecar frames.
type IPCServer struct {
	path string
	ln   net.Listener
}

// NewIPCServer creates a UDS listener at path (parent dir created on Listen).
func NewIPCServer(path string) (*IPCServer, error) {
	path = filepath.Clean(path)
	if path == "" {
		return nil, fmt.Errorf("telethon ipc: empty socket path")
	}
	return &IPCServer{path: path}, nil
}

// Listen binds the Unix socket, replacing any stale socket file.
func (s *IPCServer) Listen() error {
	if s == nil {
		return fmt.Errorf("telethon ipc: nil server")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	_ = os.Remove(s.path)
	ln, err := net.Listen("unix", s.path)
	if err != nil {
		return fmt.Errorf("telethon ipc listen %s: %w", s.path, err)
	}
	s.ln = ln
	return nil
}

// Path returns the bound socket path.
func (s *IPCServer) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Accept waits for the Python sidecar to connect.
func (s *IPCServer) Accept(ctx context.Context) (net.Conn, error) {
	if s == nil || s.ln == nil {
		return nil, fmt.Errorf("telethon ipc: not listening")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		type acceptResult struct {
			conn net.Conn
			err  error
		}
		ch := make(chan acceptResult, 1)
		go func() {
			conn, err := s.ln.Accept()
			ch <- acceptResult{conn: conn, err: err}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res := <-ch:
			if res.err != nil {
				return nil, res.err
			}
			return res.conn, nil
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// Close shuts down the listener and removes the socket file.
func (s *IPCServer) Close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.ln != nil {
		err = s.ln.Close()
		s.ln = nil
	}
	_ = os.Remove(s.path)
	return err
}
