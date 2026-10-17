package netcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestSystemCheckIgnoresEnvironmentProxyAndRejectsRedirect(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "socks5://127.0.0.1:1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cache-Control") == "" {
			t.Error("cache directive missing")
		}
		fmt.Fprint(w, `{"IP":"203.0.113.1","IsTor":false}`)
	}))
	defer server.Close()
	r, e := (Checker{url: server.URL}).Check(context.Background())
	if e != nil || r.IP != "203.0.113.1" {
		t.Fatal(r, e)
	}
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, server.URL, 302) }))
	defer redirect.Close()
	if _, e = (Checker{url: redirect.URL}).Check(context.Background()); e == nil {
		t.Fatal("followed redirect")
	}
}
func TestProxyFailureNeverFallsBackToDirect(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, `{"IP":"203.0.113.1","IsTor":false}`)
	}))
	defer server.Close()
	if _, e := (Checker{url: server.URL, Endpoint: "127.0.0.1:1"}).Check(context.Background()); e == nil {
		t.Fatal("unavailable proxy accepted")
	}
	if hits.Load() != 0 {
		t.Fatal("direct fallback reached destination")
	}
}
func TestHTTPSProxyRequestUsesRemoteHostname(t *testing.T) {
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"IP":"198.51.100.42","IsTor":true}`) }))
	defer destination.Close()
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Skip(e)
	}
	defer listener.Close()
	hostnames := make(chan string, 1)
	go func() {
		c, e := listener.Accept()
		if e != nil {
			return
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(4 * time.Second))
		hello := make([]byte, 3)
		if _, e = io.ReadFull(c, hello); e != nil {
			return
		}
		c.Write([]byte{5, 0})
		header := make([]byte, 5)
		if _, e = io.ReadFull(c, header); e != nil || header[3] != 3 {
			return
		}
		name := make([]byte, int(header[4]))
		if _, e = io.ReadFull(c, name); e != nil {
			return
		}
		port := make([]byte, 2)
		if _, e = io.ReadFull(c, port); e != nil {
			return
		}
		hostnames <- string(name)
		up, e := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port[0])<<8|int(port[1]))))
		if e != nil {
			return
		}
		defer up.Close()
		c.Write([]byte{5, 0, 0, 1, 127, 0, 0, 1, 0, 0})
		go io.Copy(up, c)
		io.Copy(c, up)
	}()
	_, port, _ := net.SplitHostPort(destination.Listener.Addr().String())
	url := "https://example.com:" + port
	transport := destination.Client().Transport.(*http.Transport).Clone()
	r, e := (Checker{url: url, Endpoint: listener.Addr().String(), transport: transport}).Check(context.Background())
	if e != nil || r.IP != "198.51.100.42" || !r.IsTor {
		t.Fatal(r, e)
	}
	select {
	case host := <-hostnames:
		if host != "example.com" {
			t.Fatal(host)
		}
	case <-time.After(time.Second):
		t.Fatal("hostname not forwarded")
	}
}
func TestRequestCancellationAndMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(time.Second):
			fmt.Fprint(w, `{"IP":"hostname","IsTor":true}`)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := (Checker{url: server.URL}).Check(ctx); e == nil {
		t.Fatal("cancellation ignored")
	}
	if _, e := (Checker{url: server.URL}).Check(context.Background()); e == nil {
		t.Fatal("invalid response accepted")
	}
}
