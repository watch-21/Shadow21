package platform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Distribution struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Family         string `json:"family"`
	PackageManager string `json:"package_manager"`
	TorUnit        string `json:"tor_unit"`
	TorUser        string `json:"tor_user"`
}

func Parse(data string) Distribution {
	fields := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		k, v, ok := strings.Cut(scanner.Text(), "=")
		if ok {
			fields[k] = strings.Trim(v, "\"'")
		}
	}
	d := Distribution{ID: fields["ID"], Name: fields["PRETTY_NAME"]}
	ids := d.ID + " " + fields["ID_LIKE"]
	for _, id := range strings.Fields(ids) {
		switch id {
		case "debian", "ubuntu", "kali", "parrot":
			d.Family = "debian"
			d.PackageManager = "apt-get"
			d.TorUnit = "tor@default.service"
			d.TorUser = "debian-tor"
		case "fedora", "rhel":
			d.Family = "fedora"
			d.PackageManager = "dnf"
			d.TorUnit = "tor.service"
			d.TorUser = "toranon"
		case "arch":
			d.Family = "arch"
			d.PackageManager = "pacman"
			d.TorUnit = "tor.service"
			d.TorUser = "tor"
		case "opensuse", "opensuse-tumbleweed", "opensuse-leap", "suse":
			d.Family = "suse"
			d.PackageManager = "zypper"
			d.TorUnit = "tor.service"
			d.TorUser = "tor"
		}
		if d.Family != "" {
			break
		}
	}
	return d
}
func Detect() Distribution { b, _ := os.ReadFile("/etc/os-release"); return Parse(string(b)) }
func Active(unit string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit).Run() == nil
}
func Preflight() error {
	d := Detect()
	if d.Family == "" {
		return fmt.Errorf("unsupported distribution")
	}
	// Non-Debian host routing stays experimental until its Tor/DNS layout is validated.
	if d.Family != "debian" {
		return fmt.Errorf("system routing is not yet validated on this distribution; use proxy-only mode")
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("systemd is required")
	}
	for _, bin := range []string{"tor", "iptables", "iptables-save", "iptables-restore", "nmcli"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("missing dependency: %s", bin)
		}
	}
	if !Active("NetworkManager.service") {
		return fmt.Errorf("active NetworkManager is required for system routing")
	}
	for _, u := range []string{"anonsurfd.service", "firewalld.service", "ufw.service"} {
		if Active(u) {
			return fmt.Errorf("conflicting active service: %s", u)
		}
	}
	for _, p := range []string{"/run/anonsurf/active", "/run/anonsurf/iptables.rules", "/var/lib/anonsurf/torrc.backup"} {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("upstream managed state must be restored first")
		}
	}
	// Refuse active VPNs because their routing and DNS ownership require a separate adapter.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nmcli", "-t", "-f", "TYPE", "connection", "show", "--active").Output()
	if err != nil {
		return fmt.Errorf("cannot inspect active connections")
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "vpn" || line == "wireguard" {
			return fmt.Errorf("active VPN conflicts with system routing")
		}
	}
	// Native nft rules outside the iptables compatibility layer are not validated here.
	if p, err := exec.LookPath("nft"); err == nil {
		out, e := exec.CommandContext(ctx, p, "list", "tables").Output()
		if e != nil {
			return fmt.Errorf("cannot inspect nftables")
		}
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) > 2 && (f[1] != "ip" || (f[2] != "filter" && f[2] != "nat" && f[2] != "mangle" && f[2] != "raw" && f[2] != "security")) {
				return fmt.Errorf("native nftables tables require a validated adapter")
			}
		}
	}
	return nil
}
