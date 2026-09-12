package telethon

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestIPCServerAccept(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "telethon.sock")

	server, err := NewIPCServer(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		client, err := net.Dial("unix", path)
		if err != nil {
			t.Error(err)
			return
		}
		_ = client.Close()
	}()

	conn, err := server.Accept(ctx)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
