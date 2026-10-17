package firewall

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/config"
	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

const (
	SavePath    = "/usr/sbin/iptables-save"
	RestorePath = "/usr/sbin/iptables-restore"
	StatePath   = "/var/lib/shadow21/firewall.owned"
	FilterChain = "SHADOW21_OUT"
	NATChain    = "SHADOW21_NAT"
)

var excludedNetworks = [...]string{"192.168.0.0/16", "172.16.0.0/12", "10.0.0.0/8", "127.0.0.0/8"}

type Manager struct {
	run        func(...string) error
	output     func(string, ...string) ([]byte, error)
	input      func(string, []byte, ...string) error
	statePath  string
	torUID     string
	killSwitch *bool
}

func (m Manager) path() string {
	if m.statePath != "" {
		return m.statePath
	}
	return StatePath
}
func (m Manager) readRules() ([]byte, error) {
	if m.output != nil {
		return m.output(SavePath)
	}
	return (command.Runner{}).Output(SavePath)
}
func (m Manager) applyRules(rules []byte) error {
	if m.input != nil {
		return m.input(RestorePath, rules, "--wait", "5", "--noflush")
	}
	return (command.Runner{}).RunWithInput(RestorePath, rules, "--wait", "5", "--noflush")
}

func (m Manager) Backup() error {
	if exists, err := fileops.Exists(m.path()); err != nil {
		return err
	} else if exists {
		return nil
	}
	out, err := m.readRules()
	if err != nil {
		return err
	}
	if strings.Contains(string(out), FilterChain) || strings.Contains(string(out), NATChain) {
		return fmt.Errorf("untracked shadow21 firewall chains; manual recovery required")
	}
	// Persistent ownership is armed before any mutation; it is not a full ruleset backup.
	return fileops.WriteAtomicWithDir(m.path(), []byte("owned-chains-v1\n"), 0600, 0700)
}
func (m Manager) Apply(routing tor.RoutingConfig) error {
	if err := validateRouting(routing); err != nil {
		return err
	}
	if _, err := os.Stat(m.path()); err != nil {
		return err
	}
	uid := m.torUID
	if uid == "" {
		u, err := user.Lookup(platform.Detect().TorUser)
		if err != nil {
			return fmt.Errorf("Tor service account unavailable")
		}
		uid = u.Uid
	}
	if _, err := strconv.Atoi(uid); err != nil {
		return err
	}
	for _, t := range []struct{ table, chain string }{{"filter", FilterChain}, {"nat", NATChain}} {
		exists := m.iptables("-t", t.table, "-S", t.chain) == nil
		if !exists {
			if err := m.iptables("-t", t.table, "-N", t.chain); err != nil {
				return err
			}
		}
	}
	// --noflush preserves unrelated chains and policies. Only our two chains are replaced.
	killSwitch := true
	if m.killSwitch != nil {
		killSwitch = *m.killSwitch
	} else {
		cfg, err := config.Load(config.SystemPath)
		if err != nil {
			return err
		}
		killSwitch = cfg.KillSwitch
	}
	rules := renderOwnedRules(routing, uid)
	if !killSwitch {
		rules = strings.Replace(rules, "-A "+FilterChain+" -j REJECT", "-A "+FilterChain+" -j RETURN", 1)
	}
	if err := m.applyRules([]byte(rules)); err != nil {
		return err
	}
	// Install the rejecting filter before installing redirection.
	for _, t := range []struct{ table, chain string }{{"filter", FilterChain}, {"nat", NATChain}} {
		if m.iptables("-t", t.table, "-C", "OUTPUT", "-j", t.chain) != nil {
			if err := m.iptables("-t", t.table, "-I", "OUTPUT", "1", "-j", t.chain); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m Manager) iptables(args ...string) error {
	if m.run != nil {
		return m.run(args...)
	}
	p, err := exec.LookPath("iptables")
	if err != nil {
		return err
	}
	return (command.Runner{}).Run(p, append([]string{"--wait", "5"}, args...)...)
}
func (m Manager) Restore() error {
	exists, err := fileops.Exists(m.path())
	if err != nil || !exists {
		return err
	}
	// Remove NAT first; the rejecting filter stays until all redirects are gone.
	for _, t := range []struct{ table, chain string }{{"nat", NATChain}, {"filter", FilterChain}} {
		if err = m.remove(t.table, t.chain); err != nil {
			return err
		}
	}
	return fileops.RemoveIfExists(m.path())
}
func (m Manager) remove(table, chain string) error {
	// Check the table can be inspected so a permission error is not mistaken for absence.
	if err := m.iptables("-t", table, "-S"); err != nil {
		return err
	}
	if m.iptables("-t", table, "-S", chain) != nil {
		return nil
	}
	for i := 0; i < 32; i++ {
		if m.iptables("-t", table, "-C", "OUTPUT", "-j", chain) != nil {
			break
		}
		if err := m.iptables("-t", table, "-D", "OUTPUT", "-j", chain); err != nil {
			return err
		}
	}
	if err := m.iptables("-t", table, "-F", chain); err != nil {
		return err
	}
	return m.iptables("-t", table, "-X", chain)
}
