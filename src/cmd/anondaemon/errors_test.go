package main

import (
	"context"
	"errors"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"strings"
	"testing"
)

func TestSessionReadinessIsNotServiceConflict(t *testing.T) {
	err := fmt.Errorf("start: %w", &proxymode.ProxyReadinessError{Cause: context.DeadlineExceeded})
	if sessionErrorCode(err) != "proxy-readiness" {
		t.Fatal(sessionErrorCode(err))
	}
	for _, row := range []struct{ message, code string }{
		{"existing Tor listener is not managed by a supported service; manual migration required", "unsupported"},
		{"existing Tor or routing service must be stopped explicitly first", "tor-conflict"},
		{"SOCKS port is occupied by an unverified process", "port-conflict"},
		{"another lifecycle operation is in progress", "busy"},
	} {
		if got := sessionErrorCode(errors.New(row.message)); got != row.code {
			t.Fatal(got, row)
		}
	}
	if got := boundedDetail(errors.New("\x1btest\x00" + strings.Repeat("x", 1500))); len(got) > 1000 || strings.ContainsAny(got, "\x1b\x00") {
		t.Fatal("unbounded or unsafe detail")
	}
}
