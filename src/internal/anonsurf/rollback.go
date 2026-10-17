package anonsurf

import (
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/firewall"
	"gitlab.com/parrotsec/packages/anonsurf/internal/ipv6"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

type startTransition struct {
	torWasActive      bool
	torStartAttempted bool
	torrcGenerated    bool
	dnsTouched        bool
	ipv6Touched       bool
	firewallTouched   bool
}

func (c Controller) failStart(operation error, state startTransition) error {
	c.reportStatus("Restoring the previous network state")
	rollback := c.rollbackStart(state)
	if rollback == nil {
		c.reportStatus("Previous network state restored")
		return operation
	}
	c.reportStatus("Network rollback failed")
	return transitionError{operation: operation, rollback: rollback}
}

func (c Controller) rollbackStart(state startTransition) error {
	var failures errorCollector
	if state.torStartAttempted {
		failures.add("stop Tor service", systemd.NewTorController().Stop())
	}
	if state.firewallTouched {
		failures.add("restore firewall", (firewall.Manager{}).Restore())
	}
	if state.ipv6Touched && failures.err() == nil {
		failures.add("restore IPv6 state", (ipv6.Manager{}).Restore())
	}
	if state.dnsTouched {
		failures.add("restore DNS settings", c.restoreDNS())
	}
	if state.torrcGenerated {
		failures.add("restore torrc", c.restoreTorrc())
	}
	if err := failures.err(); err != nil {
		return err
	}
	if state.torWasActive {
		if err := systemd.NewTorController().Start(); err != nil {
			return fmt.Errorf("restart original Tor service: %w", err)
		}
		if err := clearTorState(); err != nil {
			return err
		}
	}
	return nil
}
