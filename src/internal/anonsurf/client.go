package anonsurf

import (
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
)

const BinaryPath = "/usr/lib/shadow21/shadow21-helper"

type IdentityClient struct{}

type MonitorClient struct{}

func (IdentityClient) Change() error {
	executable, args := command.PrivilegedCommand(BinaryPath, "changeid")
	if err := (command.Runner{}).Run(executable, args...); err != nil {
		return fmt.Errorf("request privileged identity change: %w", err)
	}
	return nil
}

func (MonitorClient) Run() error {
	executable, args := command.PrivilegedCommand(BinaryPath, "monitor")
	if err := (command.Runner{}).RunInteractive(executable, args...); err != nil {
		return fmt.Errorf("open privileged Tor monitor: %w", err)
	}
	return nil
}
