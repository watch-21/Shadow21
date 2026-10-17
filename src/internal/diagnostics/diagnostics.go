package diagnostics

import (
	"context"
	"encoding/json"
	"gitlab.com/parrotsec/packages/anonsurf/internal/firewall"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Report struct {
	Distribution platform.Distribution `json:"distribution"`
	Dependencies map[string]string     `json:"dependencies"`
	Systemd      bool                  `json:"systemd"`
	ProxyReady   bool                  `json:"proxy_ready"`
	Endpoint     string                `json:"endpoint"`
	Mode         string                `json:"mode"`
	Firewall     string                `json:"firewall"`
	IPv6         string                `json:"ipv6"`
	DNS          string                `json:"dns"`
	Bootstrap    string                `json:"bootstrap"`
	Timestamp    string                `json:"timestamp"`
	Scope        string                `json:"scope"`
}

func Read(ctx context.Context, s preferences.Settings) Report {
	r := Report{Distribution: platform.Detect(), Dependencies: map[string]string{}, Endpoint: s.Endpoint, Mode: s.Mode, Firewall: "unknown", IPv6: "unknown", DNS: "unknown", Bootstrap: "unknown", Timestamp: time.Now().UTC().Format(time.RFC3339), Scope: "local-configuration-and-SOCKS-handshake; not an end-to-end leak test"}
	for _, name := range []string{"tor", "tmux", "iptables", "nft", "nmcli", "systemctl", "pkexec"} {
		p, e := exec.LookPath(name)
		if e == nil {
			r.Dependencies[name] = p
		} else {
			r.Dependencies[name] = "missing"
		}
	}
	_, e := os.Stat("/run/systemd/system")
	r.Systemd = e == nil
	r.ProxyReady = netcheck.ProbeSOCKS(ctx, s.Endpoint) == nil
	dirs, e := os.ReadDir("/proc/sys/net/ipv6/conf")
	if e == nil {
		blocked := true
		for _, d := range dirs {
			b, err := os.ReadFile(filepath.Join("/proc/sys/net/ipv6/conf", d.Name(), "disable_ipv6"))
			if err != nil {
				blocked = false
				r.IPv6 = "unknown"
				break
			}
			if strings.TrimSpace(string(b)) != "1" {
				blocked = false
			}
		}
		if blocked {
			r.IPv6 = "disabled"
		} else {
			r.IPv6 = "not-fully-disabled"
		}
	}
	if b, e := os.ReadFile("/etc/resolv.conf"); e == nil {
		r.DNS = "non-loopback-or-empty"
		found := false
		local := true
		for _, l := range strings.Split(string(b), "\n") {
			f := strings.Fields(l)
			if len(f) > 1 && f[0] == "nameserver" {
				found = true
				ip := net.ParseIP(f[1])
				if ip == nil || !ip.IsLoopback() {
					local = false
				}
			}
		}
		if found && local {
			r.DNS = "loopback-resolver; upstream route unknown"
		}
	}
	if platform.Active(r.Distribution.TorUnit) {
		var bootstrap struct {
			Progress  int    `json:"progress"`
			Timestamp string `json:"timestamp"`
		}
		if b, e := os.ReadFile("/run/shadow21-bootstrap.json"); e == nil && json.Unmarshal(b, &bootstrap) == nil {
			if stamp, e := time.Parse(time.RFC3339, bootstrap.Timestamp); e == nil && time.Since(stamp) < time.Minute && bootstrap.Progress >= 0 && bootstrap.Progress <= 100 {
				r.Bootstrap = strconv.Itoa(bootstrap.Progress) + "%"
			}
		}
	}
	// Reading firewall rules usually requires privileges; an access failure remains unknown.
	out, e := exec.CommandContext(ctx, "iptables-save").Output()
	if e == nil {
		r.Firewall = "inactive"
		text := string(out)
		if strings.Contains(text, "-A OUTPUT -j "+firewall.FilterChain) && strings.Contains(text, "-A "+firewall.FilterChain+" -j REJECT") {
			r.Firewall = "rejecting-owned-chain-present"
		}
	}
	return r
}
func Redacted(r Report) []byte {
	r.Endpoint = "loopback:[redacted]"
	r.Distribution.Name = r.Distribution.ID
	for k, v := range r.Dependencies {
		if v != "missing" {
			r.Dependencies[k] = "present"
		}
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return append(b, '\n')
}
func IdentityWait() int {
	b, e := os.ReadFile("/var/lib/shadow21/identity.timestamp")
	if e != nil {
		return -1
	}
	n, e := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if e != nil {
		return -1
	}
	wait := int(time.Until(time.Unix(n, 0).Add(15*time.Second)).Seconds()) + 1
	if wait < 0 {
		return 0
	}
	return wait
}
