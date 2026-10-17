package proxymode

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// PrepareTor explicitly starts the OS service when the user selects tmux.
// It creates no tmux session, routing rules or proxy ownership journal. The OS
// service remains running afterwards, just as with systemctl start tor.
func (m Manager) CheckPreparation(ctx context.Context) error {
	if m.ServiceUnit == "" {
		return fmt.Errorf("unsupported Tor service layout")
	}
	if e := ValidateExistingUnit(m.ServiceUnit); e != nil {
		return e
	}
	if e := m.RoutingCheck(); e != nil {
		return e
	}
	if m.IdleCheck != nil {
		if e := m.IdleCheck(); e != nil {
			return e
		}
	}
	if _, e := os.Lstat(m.statePath()); !os.IsNotExist(e) {
		return fmt.Errorf("stop or recover the owned proxy session before preparing Tor")
	}
	if m.ConfigPath != "" {
		data, e := os.ReadFile(m.ConfigPath)
		if e != nil {
			return fmt.Errorf("read Tor configuration before service start: %w", e)
		}
		if _, e = RenderConfig(string(data)); e != nil {
			return e
		}
	}
	if e := m.Preflight(ctx); e != nil {
		return e
	}
	return nil
}
func (m Manager) PrepareTor(ctx context.Context) (string, error) {
	if e := m.CheckPreparation(ctx); e != nil {
		return "", e
	}
	if e := m.command(ctx, "systemctl", "start", "tor.service"); e != nil {
		return "", fmt.Errorf("start Tor service: %w", e)
	}
	// Debian's tor.service is a wrapper; ensure the actual default instance too.
	if m.ServiceUnit != "tor.service" {
		if e := m.command(ctx, "systemctl", "start", m.ServiceUnit); e != nil {
			return "", fmt.Errorf("start Tor instance: %w", e)
		}
	}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		b, e := m.Execute(ctx, "systemctl", "show", m.ServiceUnit, "--property=ActiveState", "--value")
		if e != nil {
			return "", fmt.Errorf("cannot verify started Tor service")
		}
		state := strings.TrimSpace(string(b))
		if state == "active" {
			pid, e := m.Execute(ctx, "systemctl", "show", m.ServiceUnit, "--property=MainPID", "--value")
			n, parse := strconv.Atoi(strings.TrimSpace(string(pid)))
			if e == nil && parse == nil && n > 0 {
				if m.BindCheck == nil {
					return m.ServiceUnit, nil
				}
				if binding := m.BindCheck(ctx); binding == nil {
					return m.ServiceUnit, nil
				} else if strings.Contains(binding.Error(), "not loopback-only") {
					return "", binding
				}
				select {
				case <-ctx.Done():
					return "", fmt.Errorf("local Tor SOCKS readiness timed out")
				case <-ticker.C:
					continue
				}
			}
			return "", fmt.Errorf("Tor service has no running main process")
		}
		if state != "activating" && state != "reloading" {
			return "", fmt.Errorf("Tor service did not become active: %s", state)
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("Tor service start timed out")
		case <-ticker.C:
		}
	}
}
