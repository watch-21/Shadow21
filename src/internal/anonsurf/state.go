package anonsurf

import (
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/dns"
	"gitlab.com/parrotsec/packages/anonsurf/internal/firewall"
	"gitlab.com/parrotsec/packages/anonsurf/internal/ipv6"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

const (
	ActiveStatePath    = "/run/shadow21/active"
	DNSStatePath       = dns.StatePath
	TorStatePath       = "/run/shadow21/tor.was-active"
	stateMarkerContent = "anonsurf\n"
)

func (c Controller) markActive() error {
	return writeFile(ActiveStatePath, []byte(stateMarkerContent), 0600)
}

func (c Controller) clearActive() error {
	if err := removeFile(ActiveStatePath); err != nil {
		return fmt.Errorf("remove active state: %w", err)
	}
	return nil
}

func (c Controller) markDNSChanged() error {
	return (dns.Manager{}).MarkChanged()
}

func captureTorState(active bool) (bool, error) {
	exists, err := fileExists(TorStatePath)
	if err != nil {
		return false, fmt.Errorf("check Tor state: %w", err)
	}
	if exists {
		return true, nil
	}
	if !active {
		return false, nil
	}
	if err := writeFile(TorStatePath, []byte(stateMarkerContent), 0600); err != nil {
		return false, fmt.Errorf("save Tor state: %w", err)
	}
	return true, nil
}

func clearTorState() error {
	if err := removeFile(TorStatePath); err != nil {
		return fmt.Errorf("remove Tor state: %w", err)
	}
	return nil
}

func torWasActive() (bool, error) {
	exists, err := fileExists(TorStatePath)
	if err != nil {
		return false, fmt.Errorf("check Tor state: %w", err)
	}
	return exists, nil
}

func (c Controller) restoreDNS() error {
	path := DNSStatePath
	exists, err := fileExists(path)
	if err != nil {
		return fmt.Errorf("check DNS state: %w", err)
	}
	if !exists {
		return nil
	}
	return (dns.Manager{}).RestoreBackupOrDHCP()
}

func (c Controller) hasManagedState() (bool, error) {
	paths := []string{
		ActiveStatePath,
		firewall.StatePath,
		ipv6.StatePath,
		TorStatePath,
		tor.TorrcBackupPath,
		tor.TorrcMissingMarkerPath,
		tor.TorrcHashPath,
	}
	for _, path := range paths {
		exists, err := fileExists(path)
		if err != nil {
			return false, fmt.Errorf("check managed state %s: %w", path, err)
		}
		if exists {
			return true, nil
		}
	}
	managed, err := (dns.Manager{}).HasLifecycleState()
	if err != nil {
		return false, fmt.Errorf("check DNS state: %w", err)
	}
	return managed, nil
}

func (c Controller) Recover() error {
	c.reportStatus("Checking interrupted network state")
	active, err := fileExists(ActiveStatePath)
	if err != nil {
		return fmt.Errorf("check active state: %w", err)
	}
	if active {
		c.reportStatus("shadow21 state is active")
		return nil
	}

	managed, err := c.hasManagedState()
	if err != nil {
		return err
	}
	if !managed {
		if err := c.restoreDNS(); err != nil {
			return err
		}
		c.reportStatus("No interrupted network state found")
		return nil
	}

	var failures errorCollector
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
		c.reportStatus("Interrupted state recovery failed")
		return err
	}
	if err := c.clearActive(); err != nil {
		return err
	}
	if err := clearTorState(); err != nil {
		return err
	}
	c.reportStatus("Interrupted network state restored")
	return nil
}
