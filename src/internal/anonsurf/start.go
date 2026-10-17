package anonsurf

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/firewall"
	"gitlab.com/parrotsec/packages/anonsurf/internal/ipv6"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

func (c Controller) Start() error {
	if err := modes.RequireSystem(); err != nil {
		return err
	}
	if err := platform.Preflight(); err != nil {
		return err
	}
	activeState, err := fileExists(ActiveStatePath)
	if err != nil {
		return err
	}
	if activeState {
		return nil
	}
	managed, err := c.hasManagedState()
	if err != nil {
		return err
	}
	if managed {
		return fmt.Errorf("interrupted state exists; run recovery first")
	}
	state := startTransition{}
	c.reportStatus("Checking Tor service state")
	active, err := systemd.NewTorController().IsActiveOrActivating()
	if err != nil {
		return fmt.Errorf("check Tor service: %w", err)
	}
	state.torWasActive, err = captureTorState(active)
	if err != nil {
		return err
	}

	// This flag means "rollback is armed", not "generation succeeded",
	// so it is set before the call.
	//
	// failStart() can then restore the old file or remove the generated one;
	// if generation failed before touching disk, Restore safely does nothing.
	state.torrcGenerated = true
	c.reportStatus("Generating Tor configuration")
	if err := c.generateTorrc(); err != nil {
		return c.failStart(fmt.Errorf("generate torrc: %w", err), state)
	}

	if exists, err := fileExists(tor.TorrcPath); err != nil {
		return c.failStart(fmt.Errorf("check torrc: %w", err), state)
	} else if !exists {
		return c.failStart(ErrTorrcNotFound, state)
	}
	routing, err := c.readTorRoutingConfig()
	if err != nil {
		return c.failStart(err, state)
	}
	if err := c.startSurf(routing, &state); err != nil {
		return c.failStart(err, state)
	}

	state.torStartAttempted = true
	c.reportStatus("Starting Tor")
	if err := systemd.NewTorController().Start(); err != nil {
		return c.failStart(fmt.Errorf("start Tor service: %w", err), state)
	}
	c.reportStatus("Waiting for Tor SOCKS readiness")
	if err := tor.WaitReady(90); err != nil {
		return c.failStart(err, state)
	}
	if !routing.PreserveSystemDNS {
		c.reportStatus("Configuring Tor DNS")
		if err := c.configureDNS(&state); err != nil {
			return c.failStart(err, state)
		}
	}
	if err := c.markActive(); err != nil {
		return c.failStart(fmt.Errorf("mark daemon active: %w", err), state)
	}
	c.reportStatus("Traffic is routed through Tor")
	return nil
}

func (c Controller) startSurf(routing tor.RoutingConfig, state *startTransition) error {
	c.reportStatus("Stopping Tor before changing routes")
	if err := systemd.NewTorController().Stop(); err != nil {
		return fmt.Errorf("stop Tor service before routing: %w", err)
	}

	ipv6Manager := ipv6.Manager{}
	c.reportStatus("Disabling IPv6")
	if err := ipv6Manager.Capture(); err != nil {
		return fmt.Errorf("capture IPv6 state: %w", err)
	}
	state.ipv6Touched = true
	if err := ipv6Manager.Disable(); err != nil {
		return err
	}

	manager := firewall.Manager{}
	c.reportStatus("Applying firewall rules")
	if err := manager.Backup(); err != nil {
		return err
	}
	state.firewallTouched = true
	return manager.Apply(routing)
}

func (c Controller) readTorRoutingConfig() (tor.RoutingConfig, error) {
	data, err := readFile(tor.TorrcPath)
	if err != nil {
		return tor.RoutingConfig{}, fmt.Errorf("read torrc: %w", err)
	}
	return tor.ParseRoutingConfig(strings.NewReader(string(data)))
}
