package status

import (
	"errors"
	"fmt"
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/dns"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

type DNSState string

const (
	DNSMissing       DNSState = "missing"
	DNSEmpty         DNSState = "empty"
	DNSTor           DNSState = "tor"
	DNSTorRedirect   DNSState = "tor-redirect"
	DNSClearnet      DNSState = "clearnet"
	DNSLocalhostOnly DNSState = "localhost-only"
)

type Snapshot struct {
	AnonSurfActive bool
	TorActive      bool
	DNSState       DNSState
	DNSAddresses   []string
}

type Reader struct{}

func (Reader) Snapshot() (Snapshot, error) {
	anonsurfActive, err := systemd.NewAnonsurfdController().IsActive()
	if err != nil {
		return Snapshot{}, fmt.Errorf("read shadow21 status: %w", err)
	}
	torActive, err := systemd.NewTorController().IsActive()
	if err != nil {
		return Snapshot{}, fmt.Errorf("read Tor status: %w", err)
	}
	addresses, err := dns.ParseDNSAddresses(dns.ResolvConfPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, fmt.Errorf("read DNS status: %w", err)
	}

	return Snapshot{
		AnonSurfActive: anonsurfActive,
		TorActive:      torActive,
		DNSState:       classifyDNS(anonsurfActive, addresses, err),
		DNSAddresses:   addresses,
	}, nil
}

func classifyDNS(anonsurfActive bool, addresses []string, readErr error) DNSState {
	if errors.Is(readErr, os.ErrNotExist) {
		return DNSMissing
	}
	if len(addresses) == 0 {
		return DNSEmpty
	}
	if anonsurfActive {
		if dns.HasOnlyLocalhost(addresses) {
			return DNSTor
		}
		return DNSTorRedirect
	}
	if dns.HasOnlyLocalhost(addresses) {
		return DNSLocalhostOnly
	}
	return DNSClearnet
}
