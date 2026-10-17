package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
	"gitlab.com/parrotsec/packages/anonsurf/internal/dns"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

func showDNSStatus(printer console.Printer) error {
	addresses, err := dns.ParseDNSAddresses(dns.ResolvConfPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s was not found", errDNSUnusable, dns.ResolvConfPath)
		}
		return fmt.Errorf("read DNS addresses: %w", err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("%w: %s contains no nameservers", errDNSUnusable, dns.ResolvConfPath)
	}

	active, err := systemd.NewAnonsurfdController().IsActive()
	if err != nil {
		return fmt.Errorf("read shadow21 status: %w", err)
	}
	if !active && dns.HasOnlyLocalhost(addresses) {
		return fmt.Errorf("%w: localhost is the only DNS server while shadow21 is inactive", errDNSUnusable)
	}

	mode := "clearnet"
	if active {
		mode = "Tor DNS redirection"
		if dns.HasOnlyLocalhost(addresses) {
			mode = "Tor DNS"
		}
	}
	printer.Result("DNS status")
	printer.Resultf("  Mode:    %s\n", mode)
	printer.Resultf("  Servers: %s\n", strings.Join(addresses, ", "))
	return nil
}

func restoreDNS(manager dns.Manager, printer console.Printer) error {
	printer.Running("Restoring system DNS...")
	if err := manager.RestoreBackupOrDHCP(); err != nil {
		return fmt.Errorf("restore DNS settings: %w", err)
	}
	printer.Success("DNS settings restored.")
	return showDNSStatus(printer)
}

func writeDNSAddresses(
	manager dns.Manager,
	printer console.Printer,
	hasDHCP bool,
	addresses []string,
) error {
	printer.Running("Applying DNS settings...")
	if hasDHCP && len(addresses) == 0 {
		if err := manager.CreateDHCPOnly(); err != nil {
			return fmt.Errorf("enable DHCP DNS: %w", err)
		}
	} else if hasDHCP {
		dhcpAddresses, err := manager.DHCPAddresses()
		if err != nil {
			return fmt.Errorf("read DHCP DNS addresses: %w", err)
		}
		if err := manager.CreateAddrs(append(addresses, dhcpAddresses...)); err != nil {
			return fmt.Errorf("write DNS settings: %w", err)
		}
	} else if err := manager.CreateAddrs(addresses); err != nil {
		return fmt.Errorf("write DNS settings: %w", err)
	}

	printer.Success("DNS settings applied.")
	return showDNSStatus(printer)
}
