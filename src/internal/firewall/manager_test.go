package firewall

import (
	"errors"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnershipIsArmedBeforeAnyMutation(t *testing.T) {
	m := Manager{statePath: filepath.Join(t.TempDir(), "owned"), output: func(string, ...string) ([]byte, error) {
		return []byte("*filter\n-A OUTPUT -j UNRELATED\nCOMMIT\n"), nil
	}}
	if e := m.Backup(); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(m.path())
	if e != nil || !strings.Contains(string(b), "owned-chains") {
		t.Fatal(e, string(b))
	}
}
func TestExistingUntrackedChainsRefused(t *testing.T) {
	m := Manager{statePath: filepath.Join(t.TempDir(), "owned"), output: func(string, ...string) ([]byte, error) { return []byte(FilterChain), nil }}
	if m.Backup() == nil {
		t.Fatal("untracked chains accepted")
	}
	if _, e := os.Stat(m.path()); !os.IsNotExist(e) {
		t.Fatal("ownership marker written")
	}
}
func TestApplyNoFlushAndFilterFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned")
	os.WriteFile(path, []byte("owned"), 0600)
	steps := []string{}
	yes := true
	m := Manager{statePath: path, torUID: "109", killSwitch: &yes, run: func(args ...string) error {
		s := strings.Join(args, " ")
		steps = append(steps, s)
		if strings.Contains(s, "-C OUTPUT") {
			return errors.New("absent")
		}
		return nil
	}, input: func(_ string, data []byte, args ...string) error {
		if !strings.Contains(strings.Join(args, " "), "--noflush") {
			t.Fatal("global flush")
		}
		if strings.Contains(string(data), "UNRELATED") {
			t.Fatal("unrelated rules managed")
		}
		return nil
	}}
	r := tor.RoutingConfig{TransPort: "9040", DNSPort: "53", VirtualAddrNetwork: "10.192.0.0/10"}
	if e := m.Apply(r); e != nil {
		t.Fatal(e)
	}
	filter, nat := -1, -1
	for i, s := range steps {
		if strings.Contains(s, "-I OUTPUT 1 -j "+FilterChain) {
			filter = i
		}
		if strings.Contains(s, "-I OUTPUT 1 -j "+NATChain) {
			nat = i
		}
	}
	if filter < 0 || nat < filter {
		t.Fatal(steps)
	}
}
func TestRestoreKeepsProtectionOnNATFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned")
	os.WriteFile(path, []byte("owned"), 0600)
	m := Manager{statePath: path, run: func(args ...string) error {
		s := strings.Join(args, " ")
		if strings.Contains(s, "filter") {
			t.Fatal("filter removed after NAT failure")
		}
		return errors.New("injected failure")
	}}
	if m.Restore() == nil {
		t.Fatal("failure swallowed")
	}
	if _, e := os.Stat(path); e != nil {
		t.Fatal("recovery marker removed")
	}
}
func TestRestoreRemovesOnlyOwnedRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned")
	os.WriteFile(path, []byte("owned"), 0600)
	deleted := map[string]bool{}
	m := Manager{statePath: path, run: func(args ...string) error {
		s := strings.Join(args, " ")
		if strings.Contains(s, "-C OUTPUT") {
			if deleted[args[len(args)-1]] {
				return errors.New("absent")
			}
		}
		if strings.Contains(s, "-D OUTPUT") {
			deleted[args[len(args)-1]] = true
		}
		if strings.Contains(s, "-F ") && !strings.Contains(s, FilterChain) && !strings.Contains(s, NATChain) {
			t.Fatal(s)
		}
		return nil
	}}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("marker retained after success")
	}
}
