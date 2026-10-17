package anonsurf

import (
	"errors"
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/firewall"
	"gitlab.com/parrotsec/packages/anonsurf/internal/ipv6"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

func (c Controller) Stop() error {
	c.reportStatus("Checking managed network state")
	managed, err := c.hasManagedState()
	if err != nil {
		return err
	}
	if !managed {
		if err := c.restoreDNS(); err != nil {
			return err
		}
		c.reportStatus("shadow21 routing is inactive")
		return nil
	}
	// Recovery must not mistake a partially restored session for active routing.
	if err := c.clearActive(); err != nil {
		return err
	}
	restartTor, err := torWasActive()
	if err != nil {
		return err
	}

	var failures errorCollector
	c.reportStatus("Stopping Tor")
	failures.add("stop Tor service", systemd.NewTorController().Stop())
	c.reportStatus("Restoring firewall rules")
	failures.add("restore firewall", (firewall.Manager{}).Restore())
	c.reportStatus("Restoring DNS settings")
	failures.add("restore DNS settings", c.restoreDNS())
	c.reportStatus("Restoring Tor configuration")
	failures.add("restore torrc", c.restoreTorrc())
	c.reportStatus("Restoring IPv6")
	if failures.err() == nil {
		failures.add("restore IPv6 state", (ipv6.Manager{}).Restore())
	}
	if err := failures.err(); err != nil {
		c.reportStatus("Network restoration failed")
		return err
	}
	if restartTor && !systemd.IsSystemStopping() {
		c.reportStatus("Restarting the original Tor service")
		if err := systemd.NewTorController().Start(); err != nil {
			return fmt.Errorf("restart original Tor service: %w", err)
		}
	}
	if restartTor {
		if err := clearTorState(); err != nil {
			return err
		}
	}
	c.reportStatus("shadow21 routing is inactive")
	return nil
}

func (c Controller) restoreTorrc() error {
	if err := (tor.TorrcManager{}).Restore(); err != nil && !errors.Is(err, tor.ErrTorrcBackupNotFound) {
		return fmt.Errorf("restore torrc: %w", err)
	}
	return nil
}
