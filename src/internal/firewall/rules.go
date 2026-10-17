package firewall

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
	"net"
	"strconv"
	"strings"
)

func validateRouting(r tor.RoutingConfig) error {
	for _, p := range []string{r.DNSPort, r.TransPort} {
		n, e := strconv.Atoi(p)
		if e != nil || n < 1 || n > 65535 {
			return fmt.Errorf("invalid routing port")
		}
	}
	ip, _, err := net.ParseCIDR(r.VirtualAddrNetwork)
	if err != nil || ip.To4() == nil {
		return fmt.Errorf("invalid virtual network")
	}
	return nil
}
func renderOwnedRules(r tor.RoutingConfig, uid string) string {
	var b strings.Builder
	b.WriteString("*filter\n-F " + FilterChain + "\n")
	line := func(chain, s string) { fmt.Fprintf(&b, "-A %s %s\n", chain, s) }
	// RETURN preserves decisions made by existing firewall rules instead of bypassing them.
	line(FilterChain, "-m owner --uid-owner "+uid+" -j RETURN")
	line(FilterChain, "-p udp --sport 68 --dport 67 -j RETURN")
	for _, n := range excludedNetworks {
		line(FilterChain, "-d "+n+" -j RETURN")
	}
	line(FilterChain, "-j REJECT")
	b.WriteString("COMMIT\n*nat\n-F " + NATChain + "\n")
	line(NATChain, "-m owner --uid-owner "+uid+" -j RETURN")
	line(NATChain, "-p udp --sport 68 --dport 67 -j RETURN")
	line(NATChain, "-p udp --dport 53 -j REDIRECT --to-ports "+r.DNSPort)
	line(NATChain, "-p tcp -d "+r.VirtualAddrNetwork+" -j REDIRECT --to-ports "+r.TransPort)
	for _, n := range excludedNetworks {
		line(NATChain, "-d "+n+" -j RETURN")
	}
	line(NATChain, "-p tcp --syn -j REDIRECT --to-ports "+r.TransPort)
	b.WriteString("COMMIT\n")
	return b.String()
}
