// Package macaddr owns a temporary NetworkManager MAC override. It never
// exports connection secrets or edits the on-disk connection profile.
package macaddr

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
)

type State struct {
	Owner          int    `json:"owner"`
	Interface      string `json:"interface"`
	UUID           string `json:"uuid"`
	Property       string `json:"property"`
	PreviousPolicy string `json:"previous_policy"`
	PreviousMAC    string `json:"previous_mac"`
	AppliedMAC     string `json:"applied_mac"`
	BootID         string `json:"boot_id"`
	Phase          string `json:"phase"`
}
type Manager struct {
	Dir, Sys, BootPath string
	Execute            func(context.Context, ...string) (string, error)
	Guard              func(context.Context) error
	Random             func() (string, error)
}

var ifaceRE = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.:-]{0,14}$`)
var uuidRE = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidInterface(n string) bool {
	return n != "lo" && n != "." && n != ".." && ifaceRE.MatchString(n)
}
func New() Manager {
	return Manager{Dir: "/var/lib/shadow21/mac", Sys: "/sys/class/net", BootPath: "/proc/sys/kernel/random/boot_id", Execute: execute, Guard: guard, Random: RandomMAC}
}
func execute(ctx context.Context, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "nmcli", append([]string{"--wait", "25", "--colors", "no", "--escape", "no"}, args...)...)
	c.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C", "HOME=/root"}
	b, e := c.Output()
	if e != nil {
		detail := ""
		if exit, ok := e.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(exit.Stderr))
			if len(detail) > 500 {
				detail = detail[:500]
			}
		}
		stage := strings.Join(args[:min(2, len(args))], " ")
		return "", fmt.Errorf("NetworkManager %s failed (%v): %s", stage, e, detail)
	}
	return strings.TrimSpace(string(b)), nil
}
func guard(ctx context.Context) error {
	if e := modes.RequireSystem(); e != nil {
		return e
	}
	if e := modes.RequireProxy(); e != nil {
		return e
	}
	if _, e := execute(ctx, "-t", "-f", "STATE", "general"); e != nil {
		return e
	}
	for _, unit := range []string{"shadow21d.service", "wvpn21d.service", "anonsurfd.service"} {
		unitCheck := exec.CommandContext(ctx, "systemctl", "show", unit, "--property=LoadState,ActiveState")
		unitCheck.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
		out, err := unitCheck.Output()
		if e := inactiveService(string(out)); e != nil {
			return fmt.Errorf("%s: %w", unit, e)
		}
		// systemctl can return nonzero for a known absent unit. Only the parsed
		// not-found/inactive pair is accepted; a failed bus query stays unknown.
		if err != nil && !strings.Contains(string(out), "LoadState=not-found") {
			return fmt.Errorf("cannot establish routing service state: %s", unit)
		}
	}
	active, e := execute(ctx, "-t", "-f", "TYPE", "connection", "show", "--active")
	if e != nil {
		return e
	}
	for _, kind := range strings.Fields(active) {
		if kind == "vpn" || kind == "wireguard" {
			return fmt.Errorf("stop active VPN connections before changing MAC")
		}
	}
	return nil
}
func RandomMAC() (string, error) {
	b := make([]byte, 6)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	b[0] = (b[0] | 2) & 0xfe
	return net.HardwareAddr(b).String(), nil
}

// Canonicalize only literal MAC addresses; symbolic NetworkManager policies
// retain their meaning and are never treated as application-owned overrides.
func canonicalPolicy(s string) string {
	s = strings.TrimSpace(s)
	if s == "--" {
		return ""
	}
	if b, e := net.ParseMAC(s); e == nil && len(b) == 6 {
		return b.String()
	}
	return s
}
func validMAC(s string) bool {
	b, e := net.ParseMAC(s)
	return e == nil && len(b) == 6 && b[0]&1 == 0 && s != "00:00:00:00:00:00"
}
func (m Manager) path(n string) string { return filepath.Join(m.Dir, n+".json") }
func (m Manager) address(n string) (string, error) {
	if !ValidInterface(n) {
		return "", fmt.Errorf("invalid network interface")
	}
	b, e := os.ReadFile(filepath.Join(m.Sys, n, "type"))
	if e != nil || strings.TrimSpace(string(b)) != "1" {
		return "", fmt.Errorf("unsupported interface type")
	}
	if _, e = os.Stat(filepath.Join(m.Sys, n, "device")); e != nil {
		return "", fmt.Errorf("only physical Ethernet or Wi-Fi adapters are supported")
	}
	b, e = os.ReadFile(filepath.Join(m.Sys, n, "address"))
	s := strings.ToLower(strings.TrimSpace(string(b)))
	if e != nil || !validMAC(s) {
		return "", fmt.Errorf("current MAC is unavailable")
	}
	return s, nil
}
func (m Manager) boot() (string, error) {
	b, e := os.ReadFile(m.BootPath)
	s := strings.TrimSpace(string(b))
	if e != nil || !uuidRE.MatchString(s) {
		return "", fmt.Errorf("boot identity is unavailable")
	}
	return s, nil
}
func (m Manager) save(s State) error {
	b, e := json.Marshal(s)
	if e != nil {
		return e
	}
	return fileops.WriteAtomicWithDir(m.path(s.Interface), append(b, '\n'), 0600, 0700)
}
func (m Manager) read(n string) (State, error) {
	var s State
	if !ValidInterface(n) {
		return s, fmt.Errorf("invalid network interface")
	}
	info, e := os.Lstat(m.path(n))
	if e != nil {
		return s, e
	}
	if !info.Mode().IsRegular() {
		return s, fmt.Errorf("MAC journal must be a regular file")
	}
	b, e := os.ReadFile(m.path(n))
	if e != nil {
		return s, e
	}
	if json.Unmarshal(b, &s) != nil || s.Interface != n || s.Owner < 0 || !uuidRE.MatchString(s.UUID) || !uuidRE.MatchString(s.BootID) || !validMAC(s.PreviousMAC) || !validMAC(s.AppliedMAC) || (s.Property != "802-3-ethernet.cloned-mac-address" && s.Property != "802-11-wireless.cloned-mac-address") || (s.Phase != "applying" && s.Phase != "active" && s.Phase != "restoring" && s.Phase != "failed") {
		return s, fmt.Errorf("invalid MAC recovery journal")
	}
	s.PreviousMAC = canonicalPolicy(s.PreviousMAC)
	s.AppliedMAC = canonicalPolicy(s.AppliedMAC)
	s.PreviousPolicy = canonicalPolicy(s.PreviousPolicy)
	return s, nil
}
func (m Manager) query(ctx context.Context, field string, args ...string) (string, error) {
	return m.Execute(ctx, append([]string{"-g", field}, args...)...)
}
func (m Manager) activeUUID(ctx context.Context, n string) (string, error) {
	s, e := m.query(ctx, "GENERAL.CON-UUID", "device", "show", n)
	if s == "--" {
		s = ""
	}
	if e != nil {
		return "", e
	}
	if s != "" && !uuidRE.MatchString(s) {
		return "", fmt.Errorf("invalid active connection UUID")
	}
	return s, nil
}
func (m Manager) Apply(ctx context.Context, uid int, n string) error {
	if !ValidInterface(n) {
		return fmt.Errorf("invalid network interface")
	}
	if e := m.Guard(ctx); e != nil {
		return e
	}
	if _, e := os.Lstat(m.path(n)); !os.IsNotExist(e) {
		saved, err := m.read(n)
		if err != nil {
			return err
		}
		if uid != 0 && saved.Owner != uid {
			return fmt.Errorf("MAC operation belongs to another user")
		}
		if saved.Phase != "active" {
			return fmt.Errorf("recover the interrupted MAC operation before changing MAC")
		}
		// Restore the original settings through the guarded recovery path first.
		// A fresh transaction then retains the original address, never the last
		// generated address, as its restoration target.
		var next string
		for attempt := 0; attempt < 8; attempt++ {
			next, err = m.Random()
			if err != nil {
				return err
			}
			next = canonicalPolicy(next)
			if !validMAC(next) {
				return fmt.Errorf("invalid generated MAC address")
			}
			if next != saved.AppliedMAC && next != saved.PreviousMAC {
				break
			}
			next = ""
		}
		if next == "" {
			return fmt.Errorf("could not generate a different MAC address")
		}
		if err = m.Restore(ctx, uid, n); err != nil {
			return err
		}
		m.Random = func() (string, error) { return next, nil }
		return m.Apply(ctx, uid, n)
	}
	before, e := m.address(n)
	if e != nil {
		return e
	}
	uuid, e := m.activeUUID(ctx, n)
	if e != nil || uuid == "" {
		return fmt.Errorf("an active NetworkManager connection is required")
	}
	kind, e := m.query(ctx, "connection.type", "connection", "show", "uuid", uuid)
	if e != nil {
		return e
	}
	property := ""
	switch kind {
	case "802-3-ethernet":
		property = "802-3-ethernet.cloned-mac-address"
	case "802-11-wireless":
		property = "802-11-wireless.cloned-mac-address"
	default:
		return fmt.Errorf("unsupported NetworkManager connection type")
	}
	controller, e := m.query(ctx, "connection.master", "connection", "show", "uuid", uuid)
	if e != nil {
		return e
	}
	if controller != "" && controller != "--" {
		return fmt.Errorf("bridge and bond ports are unsupported")
	}
	multi, e := m.query(ctx, "connection.multi-connect", "connection", "show", "uuid", uuid)
	if e != nil {
		return e
	}
	fields := strings.Fields(multi)
	if len(fields) == 0 || (fields[0] != "0" && fields[0] != "1") {
		return fmt.Errorf("shared multi-connect profiles are unsupported")
	}
	old, e := m.query(ctx, property, "connection", "show", "uuid", uuid)
	if e != nil {
		return e
	}
	old = canonicalPolicy(old)
	boot, e := m.boot()
	if e != nil {
		return e
	}
	next, e := m.Random()
	if e != nil {
		return e
	}
	next = canonicalPolicy(next)
	if !validMAC(next) || next == before {
		return fmt.Errorf("invalid generated MAC address")
	}
	s := State{Owner: uid, Interface: n, UUID: uuid, Property: property, PreviousPolicy: old, PreviousMAC: before, AppliedMAC: next, BootID: boot, Phase: "applying"}
	if e = m.save(s); e != nil {
		return e
	}
	if _, e = m.Execute(ctx, "connection", "modify", "--temporary", "uuid", uuid, property, next); e == nil {
		_, e = m.Execute(ctx, "device", "disconnect", n)
		if e == nil {
			_, e = m.Execute(ctx, "connection", "up", "uuid", uuid, "ifname", n)
		}
	}
	if e == nil {
		var got string
		got, e = m.address(n)
		if e == nil && got != next {
			e = fmt.Errorf("MAC change was not observed")
		}
	}
	if e != nil {
		recovery, cancel := context.WithTimeout(context.Background(), 80*time.Second)
		defer cancel()
		if r := m.restore(recovery, &s); r != nil {
			s.Phase = "failed"
			_ = m.save(s)
			return errors.Join(e, fmt.Errorf("MAC rollback needs recovery: %w", r))
		}
		return e
	}
	s.Phase = "active"
	return m.save(s)
}
func (m Manager) Restore(ctx context.Context, uid int, n string) error {
	if !ValidInterface(n) {
		return fmt.Errorf("invalid network interface")
	}
	if e := m.Guard(ctx); e != nil {
		return e
	}
	s, e := m.read(n)
	if os.IsNotExist(e) {
		return fmt.Errorf("no saved MAC operation for this interface")
	}
	if e != nil {
		return e
	}
	if uid != 0 && s.Owner != uid {
		return fmt.Errorf("MAC operation belongs to another user")
	}
	if e = m.restore(ctx, &s); e != nil {
		s.Phase = "failed"
		_ = m.save(s)
	}
	return e
}
func (m Manager) restore(ctx context.Context, s *State) error {
	boot, e := m.boot()
	if e != nil {
		return e
	}
	policy, e := m.query(ctx, s.Property, "connection", "show", "uuid", s.UUID)
	if e != nil {
		return e
	}
	policy = canonicalPolicy(policy)
	if boot != s.BootID {
		// --temporary changes vanish on daemon/reboot recovery. Never rewrite a
		// newly booted connection to reproduce an old random runtime address.
		if policy != s.PreviousPolicy {
			return fmt.Errorf("MAC policy changed externally after reboot")
		}
		return fileops.RemoveIfExists(m.path(s.Interface))
	}
	if policy != s.AppliedMAC && policy != s.PreviousPolicy && policy != s.PreviousMAC {
		return fmt.Errorf("MAC policy changed externally; preserving it")
	}
	active, e := m.activeUUID(ctx, s.Interface)
	if e != nil {
		return e
	}
	if active != "" && active != s.UUID {
		return fmt.Errorf("another connection is active; preserving it")
	}
	s.Phase = "restoring"
	if e = m.save(*s); e != nil {
		return e
	}
	if _, e = m.Execute(ctx, "connection", "modify", "--temporary", "uuid", s.UUID, s.Property, s.PreviousMAC); e != nil {
		return e
	}
	if active != "" {
		if _, e = m.Execute(ctx, "device", "disconnect", s.Interface); e != nil {
			return e
		}
	}
	if _, e = m.Execute(ctx, "connection", "up", "uuid", s.UUID, "ifname", s.Interface); e != nil {
		return e
	}
	observed, e := m.address(s.Interface)
	if e != nil {
		return e
	}
	if observed != s.PreviousMAC {
		return fmt.Errorf("previous MAC was not restored")
	}
	if _, e = m.Execute(ctx, "connection", "modify", "--temporary", "uuid", s.UUID, s.Property, s.PreviousPolicy); e != nil {
		return e
	}
	return fileops.RemoveIfExists(m.path(s.Interface))
}
func (m Manager) RestoreAll(ctx context.Context, uid int) error {
	paths, e := filepath.Glob(filepath.Join(m.Dir, "*.json"))
	if e != nil {
		return e
	}
	for _, p := range paths {
		n := strings.TrimSuffix(filepath.Base(p), ".json")
		if e = m.Restore(ctx, uid, n); e != nil {
			return e
		}
	}
	return nil
}

func (m Manager) RecoverInterrupted(ctx context.Context) error {
	paths, e := filepath.Glob(filepath.Join(m.Dir, "*.json"))
	if e != nil {
		return e
	}
	boot, e := m.boot()
	if e != nil {
		return e
	}
	for _, p := range paths {
		n := strings.TrimSuffix(filepath.Base(p), ".json")
		s, e := m.read(n)
		if e != nil {
			return e
		}
		if s.BootID != boot || s.Phase != "active" {
			if e = m.Restore(ctx, 0, n); e != nil {
				return e
			}
		}
	}
	return nil
}
func (m Manager) RequireReady() error {
	paths, e := filepath.Glob(filepath.Join(m.Dir, "*.json"))
	if e != nil {
		return e
	}
	for _, p := range paths {
		s, e := m.read(strings.TrimSuffix(filepath.Base(p), ".json"))
		if e != nil {
			return e
		}
		if s.Phase != "active" {
			return fmt.Errorf("recover the interrupted MAC operation before starting a connection")
		}
	}
	return nil
}

func inactiveService(text string) error {
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			fields[k] = strings.TrimSpace(v)
		}
	}
	if fields["LoadState"] == "not-found" && fields["ActiveState"] == "inactive" {
		return nil
	}
	if fields["LoadState"] != "loaded" && fields["LoadState"] != "masked" {
		return fmt.Errorf("cannot establish routing service state")
	}
	switch fields["ActiveState"] {
	case "inactive", "failed":
		return nil
	case "active", "activating", "deactivating", "reloading":
		return fmt.Errorf("stop both connection modes before changing MAC")
	default:
		return fmt.Errorf("cannot establish routing service state")
	}
}
