package netcheck

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestSOCKSRemoteHostname(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Skipf("loopback unavailable: %v", e)
	}
	defer l.Close()
	done := make(chan string, 1)
	go func() {
		c, e := l.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(3 * time.Second))
		b := make([]byte, 3)
		io.ReadFull(c, b)
		c.Write([]byte{5, 0})
		h := make([]byte, 5)
		io.ReadFull(c, h)
		if h[3] != 3 {
			done <- "not remote DNS"
			return
		}
		name := make([]byte, int(h[4])+2)
		io.ReadFull(c, name)
		done <- string(name[:len(name)-2])
		c.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 1, 1})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, e := DialSOCKS(ctx, l.Addr().String(), "test.invalid:443")
	if e != nil {
		t.Fatal(e)
	}
	c.Close()
	if got := <-done; got != "test.invalid" {
		t.Fatal(got)
	}
}
func TestMalformedCheckResponse(t *testing.T) {
	for _, b := range []string{`{}`, `{"IP":"hostname","IsTor":true}`, `{"IP":"127.0.0.1"}`, `not json`} {
		if _, e := parseTorCheckResponse(b); e == nil {
			t.Fatal("bad response accepted")
		}
	}
}

func TestConnectReplyResetsGreetingDeadline(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	client.SetDeadline(time.Now().Add(-time.Second))
	go func() {
		header := make([]byte, 5)
		if _, err := io.ReadFull(server, header); err != nil {
			return
		}
		io.CopyN(io.Discard, server, int64(header[4])+2)
		time.Sleep(10 * time.Millisecond)
		server.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := connectSOCKS(ctx, client, "example.invalid", 443)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}
func TestConnectReplyCancellationClosesSocket(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	go io.Copy(io.Discard, server)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)
	_, err := connectSOCKS(ctx, client, "example.invalid", 443)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
