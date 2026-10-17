package netcheck

import (
	"context"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"io"
	"net"
	"strconv"
	"time"
)

func ProbeSOCKS(ctx context.Context, endpoint string) error {
	c, err := socksConnection(ctx, endpoint)
	if err != nil {
		return err
	}
	return c.Close()
}
func socksConnection(ctx context.Context, endpoint string) (net.Conn, error) {
	if err := preferences.ValidateEndpoint(endpoint); err != nil {
		return nil, err
	}
	c, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return nil, err
	}
	if err = c.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		c.Close()
		return nil, err
	}
	if _, err = c.Write([]byte{5, 1, 0}); err != nil {
		c.Close()
		return nil, err
	}
	reply := make([]byte, 2)
	if _, err = io.ReadFull(c, reply); err != nil || reply[0] != 5 || reply[1] != 0 {
		c.Close()
		return nil, fmt.Errorf("SOCKS5 handshake failed")
	}
	return c, nil
}
func DialSOCKS(ctx context.Context, endpoint, address string) (net.Conn, error) {
	host, port, e := net.SplitHostPort(address)
	if e != nil {
		return nil, e
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 || len(host) > 255 {
		return nil, fmt.Errorf("invalid SOCKS destination")
	}
	c, e := socksConnection(ctx, endpoint)
	if e != nil {
		return nil, e
	}
	return connectSOCKS(ctx, c, host, n)
}

// The initial greeting timeout must not also cap Tor's circuit/remote-DNS
// response. This phase gets a fresh deadline, bounded by request cancellation.
func connectSOCKS(ctx context.Context, c net.Conn, host string, n int) (net.Conn, error) {
	fail := func(err error) (net.Conn, error) {
		c.Close()
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return nil, fmt.Errorf("SOCKS destination connection: %w", err)
	}
	deadline := time.Now().Add(20 * time.Second)
	if end, ok := ctx.Deadline(); ok && end.Before(deadline) {
		deadline = end
	}
	if e := c.SetDeadline(deadline); e != nil {
		return fail(e)
	}
	stopCancel := context.AfterFunc(ctx, func() { c.Close() })
	defer stopCancel()
	var e error

	req := append([]byte{5, 1, 0, 3, byte(len(host))}, []byte(host)...)
	req = append(req, byte(n>>8), byte(n))
	if _, e = c.Write(req); e != nil {
		return fail(e)
	}
	header := make([]byte, 4)
	if _, e = io.ReadFull(c, header); e != nil {
		return fail(e)
	}
	if header[0] != 5 || header[1] != 0 || header[2] != 0 {
		return fail(fmt.Errorf("SOCKS connect rejected"))
	}
	size := 0
	switch header[3] {
	case 1:
		size = 4
	case 4:
		size = 16
	case 3:
		b := []byte{0}
		if _, e = io.ReadFull(c, b); e != nil {
			return fail(e)
		}
		size = int(b[0])
	default:
		return fail(fmt.Errorf("invalid SOCKS address"))
	}
	if _, e = io.CopyN(io.Discard, c, int64(size+2)); e != nil {
		return fail(e)
	}
	if !stopCancel() {
		return fail(ctx.Err())
	}
	if e = c.SetDeadline(time.Time{}); e != nil {
		return fail(e)
	}
	return c, nil
}
