package modes

import (
	"fmt"
	"os"
)

const ProxyStatePath = "/var/lib/shadow21/proxy-state.json"
const PublicPath = "/run/shadow21-mode.json"

var SystemArtifacts = []string{
	"/run/shadow21/active", "/var/lib/shadow21/firewall.owned", "/var/lib/shadow21/ipv6.state",
	"/run/shadow21/tor.was-active", "/run/anonsurf/active", "/run/anonsurf/iptables.rules", "/var/lib/anonsurf/torrc.backup", "/var/lib/shadow21/torrc.backup", "/var/lib/shadow21/torrc.missing", "/var/lib/shadow21/torrc.managed-hashes", "/var/lib/shadow21/dns.changed",
}

func RequireAbsent(paths []string) error {
	for _, p := range paths {
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("conflicting mode or unfinished recovery; stop/recover it first")
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot establish mode exclusivity: %w", err)
		}
	}
	return nil
}

var LegacyArtifacts = []string{"/run/wvpn21/active", "/run/wvpn21-mode.json", "/var/lib/wvpn21/proxy-state.json", "/var/lib/wvpn21/firewall.owned", "/var/lib/wvpn21/torrc.backup", "/var/lib/wvpn21/dns.changed"}

func RequireSystem() error {
	return RequireAbsent(append([]string{ProxyStatePath, PublicPath}, LegacyArtifacts...))
}
func RequireProxy() error { return RequireAbsent(append(SystemArtifacts, LegacyArtifacts...)) }
