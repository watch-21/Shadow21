package main

import (
	"context"
	"errors"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"strings"
	"testing"
)

func TestRecoveryBeforeLifecycleLockAndTorStart(t *testing.T) {
	held := false
	recovered := false
	checks := 0
	started := false
	m := proxymode.Manager{ServiceUnit: "tor.service", StateDir: t.TempDir(), RoutingCheck: func() error { checks++; return nil }, IdleCheck: func() error { return nil }, Preflight: func(context.Context) error { return nil }}
	m.Execute = func(_ context.Context, p string, a ...string) ([]byte, error) {
		if !held || !recovered {
			t.Fatal("Tor started before recovery/lock")
		}
		started = true
		if strings.Contains(strings.Join(a, " "), "MainPID") {
			return []byte("123"), nil
		}
		return []byte("active"), nil
	}
	unit, e := prepareSelection(context.Background(), m, func(context.Context) error {
		if held {
			t.Fatal("Recovery blocked on caller lock")
		}
		recovered = true
		return nil
	}, func() (func(), error) { held = true; return func() { held = false }, nil })
	if e != nil || unit != "tor.service" || held || checks != 2 || !started {
		t.Fatal(unit, e, held, checks, started)
	}
	started = false
	_, e = prepareSelection(context.Background(), m, func(context.Context) error { return errors.New("recovery failure") }, func() (func(), error) { t.Fatal("Lock after failed recovery"); return nil, nil })
	if e == nil || started {
		t.Fatal("Recovery failure allowed Tor start")
	}
}
