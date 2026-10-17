package firewall

import (
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
	"strings"
	"testing"
)

func TestOwnedRulesHaveNoGlobalMutation(t *testing.T) {
	r := tor.RoutingConfig{TransPort: "9040", DNSPort: "53", VirtualAddrNetwork: "10.192.0.0/10"}
	s := renderOwnedRules(r, "109")
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "-F ") && line != "-F "+FilterChain && line != "-F "+NATChain {
			t.Fatal(line)
		}
		if strings.HasPrefix(line, "-A ") && !strings.HasPrefix(line, "-A "+FilterChain+" ") && !strings.HasPrefix(line, "-A "+NATChain+" ") {
			t.Fatal(line)
		}
		if strings.HasPrefix(line, ":") || strings.HasPrefix(line, "-P") {
			t.Fatal("policy changed")
		}
	}
	if !strings.Contains(s, "-j REJECT") || !strings.Contains(s, "--uid-owner 109 -j RETURN") {
		t.Fatal("missing protection")
	}
	if strings.Contains(s, "-p udp -j REDIRECT") {
		t.Fatal("unsupported UDP redirected")
	}
}
func TestRejectInjectedRouting(t *testing.T) {
	r := tor.RoutingConfig{TransPort: "9040\n-F OUTPUT", DNSPort: "53", VirtualAddrNetwork: "10.192.0.0/10"}
	if validateRouting(r) == nil {
		t.Fatal("injection accepted")
	}
}
