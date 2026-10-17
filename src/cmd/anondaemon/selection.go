package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/macaddr"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"os"
	"os/exec"
	"time"
)

func prepareTorSelection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if e := macaddr.New().RequireReady(); e != nil {
		return e
	}
	manager := proxymode.New()
	unit, e := prepareSelection(ctx, manager, func(ctx context.Context) error {
		// This service itself needs the lifecycle lock. Complete it before taking
		// that lock, otherwise the Tor dependency can fail on first installation.
		if e := exec.CommandContext(ctx, "systemctl", "start", "shadow21-recovery.service").Run(); e != nil {
			return fmt.Errorf("prepare recovery service before Tor start: %w", e)
		}
		return nil
	}, func() (func(), error) { return modes.Lock("/run/shadow21/lifecycle.lock") })
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"prepared": true, "unit": unit})
}
func prepareSelection(ctx context.Context, m proxymode.Manager, recoverService func(context.Context) error, lock func() (func(), error)) (string, error) {
	if e := m.CheckPreparation(ctx); e != nil {
		return "", e
	}
	if e := recoverService(ctx); e != nil {
		return "", e
	}
	release, e := lock()
	if e != nil {
		return "", e
	}
	defer release()
	// Recheck after acquiring the lock: another CLI may have started meanwhile.
	return m.PrepareTor(ctx)
}
