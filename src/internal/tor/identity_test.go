package tor

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

func TestIdentityUsesAuthenticatedNEWNYM(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	done := make(chan error, 1)
	go func() {
		defer b.Close()
		r := bufio.NewReader(b)
		line, _ := r.ReadString('\n')
		if line != "AUTHENTICATE abc\r\n" {
			done <- net.ErrClosed
			return
		}
		b.Write([]byte("250 OK\r\n"))
		line, _ = r.ReadString('\n')
		if line != "SIGNAL NEWNYM\r\n" {
			done <- net.ErrClosed
			return
		}
		b.Write([]byte("250 OK\r\n"))
		r.ReadString('\n')
		done <- nil
	}()
	if e := requestNewIdentity(a, "abc"); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}
func TestControlFailureNotSuccess(t *testing.T) {
	if readControlOK(bufio.NewReader(strings.NewReader("515 Authentication failed\r\n")), "auth") == nil {
		t.Fatal("failure accepted")
	}
}
func TestTorrcRejectsMultilineBridge(t *testing.T) {
	o := TorrcOptions{PlaintextPorts: "23", BridgeAddress: "1.2.3.4:443\nSocksPort 0"}
	if _, e := RenderTorrc(o); e == nil {
		t.Fatal("injection accepted")
	}
}
