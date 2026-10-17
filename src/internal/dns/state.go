package dns

import (
	"errors"
	"fmt"
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

const StatePath = "/var/lib/shadow21/dns.changed"

type dnsState string

const (
	stateUntracked          dnsState = ""
	stateManaged            dnsState = "anonsurf\n"
	stateRestored           dnsState = "restored\n"
	stateRestoredStandalone dnsState = "restored-standalone\n"
)

func (state dnsState) isRestored() bool {
	return state == stateRestored || state == stateRestoredStandalone
}

func readDNSState() (dnsState, error) {
	content, err := os.ReadFile(StatePath)
	if errors.Is(err, os.ErrNotExist) {
		return stateUntracked, nil
	}
	if err != nil {
		return stateUntracked, fmt.Errorf("read DNS state: %w", err)
	}
	state := dnsState(content)
	switch state {
	case stateManaged, stateRestored, stateRestoredStandalone:
		return state, nil
	default:
		return stateUntracked, fmt.Errorf("invalid DNS state %q", content)
	}
}

func writeDNSState(state dnsState) error {
	if err := fileops.WriteAtomicWithDir(StatePath, []byte(state), 0600, 0700); err != nil {
		return fmt.Errorf("write DNS state: %w", err)
	}
	return nil
}

func (m Manager) MarkChanged() error {
	if err := m.finishPendingRestore(); err != nil {
		return err
	}
	return writeDNSState(stateManaged)
}

func (Manager) IsManaged() (bool, error) {
	state, err := readDNSState()
	return state == stateManaged, err
}

func (Manager) HasLifecycleState() (bool, error) {
	state, err := readDNSState()
	return state == stateManaged || state == stateRestored, err
}
