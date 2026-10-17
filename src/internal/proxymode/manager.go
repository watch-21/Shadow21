package proxymode

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

const Endpoint = "127.0.0.1:9050"
const TorUnit = "shadow21-proxy-tor.service"

var namePattern = regexp.MustCompile(`^shadow21-[a-zA-Z0-9_-]{1,48}$`)

func ValidateName(n string) error {
	if n != "0" && n != "tor" && !namePattern.MatchString(n) {
		return fmt.Errorf("invalid shadow21 session name")
	}
	return nil
}

type State struct {
	Owner                 int      `json:"owner"`
	Sessions              []string `json:"sessions"`
	Phase                 string   `json:"phase"`
	ExistingTorUnit       string   `json:"existing_tor_unit,omitempty"`
	TorStarted            bool     `json:"tor_started"`
	ConfigChanged         bool     `json:"config_changed"`
	OriginalExists        bool     `json:"original_exists"`
	OriginalMode          uint32   `json:"original_mode"`
	OriginalUID           int      `json:"original_uid"`
	OriginalGID           int      `json:"original_gid"`
	OriginalOwnerRecorded bool     `json:"original_owner_recorded"`
	OriginalHash          string   `json:"original_hash"`
	ManagedHash           string   `json:"managed_hash"`
}
type Public struct {
	Owner           int      `json:"owner"`
	Sessions        []string `json:"sessions"`
	Phase           string   `json:"phase"`
	Mode            string   `json:"mode"`
	ExistingTorUnit string   `json:"existing_tor_unit,omitempty"`
}
type Manager struct {
	ServiceUnit                                  string
	IdleCheck                                    func() error
	Report                                       func(string)
	StateDir, RuntimeDir, ConfigPath, PublicPath string
	Execute                                      func(context.Context, string, ...string) ([]byte, error)
	Preflight                                    func(context.Context) error
	Ready                                        func(context.Context) error
	BindCheck                                    func(context.Context) error
	RoutingCheck                                 func() error
	Existing                                     func(context.Context) (string, error)
}

func New() Manager {
	return Manager{ServiceUnit: platform.Detect().TorUnit, IdleCheck: modes.RequireSystem, StateDir: "/var/lib/shadow21", RuntimeDir: "/run/shadow21", ConfigPath: "/etc/tor/torrc", PublicPath: modes.PublicPath, Execute: run, Preflight: preflight, Ready: ready, BindCheck: bindings, RoutingCheck: modes.RequireProxy, Existing: existingProxy}
}
func run(ctx context.Context, p string, args ...string) ([]byte, error) {
	c := exec.CommandContext(ctx, p, args...)
	c.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "HOME=/root", "TERM=xterm-256color", "LANG=C.UTF-8"}
	return c.CombinedOutput()
}
func (m Manager) statePath() string      { return filepath.Join(m.StateDir, "proxy-state.json") }
func (m Manager) backupPath() string     { return filepath.Join(m.StateDir, "proxy-torrc.backup") }
func (m Manager) socket(n string) string { return filepath.Join(m.RuntimeDir, n+".sock") }
func unit(n string) string               { return "shadow21-tmux-" + n + ".service" }
func hash(b []byte) string               { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func (m Manager) read() (State, error) {
	var s State
	b, e := os.ReadFile(m.statePath())
	if e != nil {
		return s, e
	}
	if e = json.Unmarshal(b, &s); e != nil {
		return s, e
	}
	if s.OriginalMode > 0777 || (s.OriginalOwnerRecorded && (s.OriginalUID < 0 || s.OriginalGID < 0)) || s.Owner < 0 || (s.Phase != "starting" && s.Phase != "active" && s.Phase != "stopping" && s.Phase != "failed") {
		return s, fmt.Errorf("invalid proxy recovery state")
	}
	if e = ValidateExistingUnit(s.ExistingTorUnit); e != nil {
		return s, e
	}
	for _, n := range s.Sessions {
		if e = ValidateName(n); e != nil {
			return s, e
		}
	}
	return s, nil
}
func (m Manager) save(s State) error {
	b, e := json.Marshal(s)
	if e != nil {
		return e
	}
	if e = fileops.WriteAtomicWithDir(m.statePath(), append(b, '\n'), 0600, 0700); e != nil {
		return e
	}
	p := Public{Owner: s.Owner, Sessions: s.Sessions, Phase: s.Phase, Mode: "proxy", ExistingTorUnit: s.ExistingTorUnit}
	b, _ = json.Marshal(p)
	return fileops.WriteAtomic(m.PublicPath, append(b, '\n'), 0644)
}
func (m Manager) own(s State, uid int) error {
	if uid != 0 && s.Owner != uid {
		return fmt.Errorf("proxy mode belongs to another user")
	}
	return nil
}
func (m Manager) command(ctx context.Context, p string, a ...string) error {
	b, e := m.Execute(ctx, p, a...)
	if e != nil {
		if p == "tor" {
			return fmt.Errorf("Tor configuration validation failed; inspect the configuration locally")
		}
		return fmt.Errorf("%s operation failed: %s", filepath.Base(p), strings.TrimSpace(string(b)))
	}
	return nil
}
func (m Manager) tmux(ctx context.Context, n string, a ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return m.command(ctx, "tmux", append([]string{"-N", "-f", "/dev/null", "-S", m.socket(n)}, a...)...)
}
func (m Manager) active(ctx context.Context, u string) (bool, error) {
	b, e := m.Execute(ctx, "systemctl", "show", u, "--property=ActiveState", "--value")
	if e != nil {
		if missingUnit(string(b)) {
			return false, nil
		}
		return false, fmt.Errorf("cannot inspect owned service")
	}
	switch strings.TrimSpace(string(b)) {
	case "inactive", "failed":
		return false, nil
	case "active", "activating", "deactivating", "reloading":
		return true, nil
	default:
		return false, fmt.Errorf("unknown service state")
	}
}
func (m Manager) configure(ctx context.Context, s *State) error {
	info, e := os.Lstat(m.ConfigPath)
	var original []byte
	if os.IsNotExist(e) {
		s.OriginalExists = false
		s.OriginalMode = 0600
		s.OriginalUID = 0
		s.OriginalGID = 0
		s.OriginalOwnerRecorded = true
	} else if e != nil {
		return e
	} else {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("Tor configuration must be a regular file")
		}
		s.OriginalExists = true
		s.OriginalMode = uint32(info.Mode().Perm())
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			s.OriginalUID = int(stat.Uid)
			s.OriginalGID = int(stat.Gid)
			s.OriginalOwnerRecorded = true
		} else {
			return fmt.Errorf("cannot record Tor configuration ownership")
		}
		original, e = os.ReadFile(m.ConfigPath)
		if e != nil {
			return e
		}
	}
	rendered, e := RenderConfig(string(original))
	if e != nil {
		return e
	}
	s.OriginalHash = hash(original)
	s.ManagedHash = hash([]byte(rendered))
	if e = fileops.WriteAtomic(m.backupPath(), original, 0600); e != nil {
		return e
	}
	candidate := filepath.Join(m.RuntimeDir, "proxy-torrc-check")
	if e = fileops.WriteAtomic(candidate, []byte(rendered), 0600); e != nil {
		return e
	}
	defer os.Remove(candidate)
	if e = m.command(ctx, "tor", append([]string{"--verify-config"}, torArgs(candidate)...)...); e != nil {
		return e
	}
	// Arm restoration before writing, including a crash between journal and rename.
	s.ConfigChanged = true
	if e = m.save(*s); e != nil {
		return e
	}
	return fileops.WriteAtomicWithDir(m.ConfigPath, []byte(rendered), 0600, 0755)
}
func torArgs(config string) []string {
	return []string{"-f", config, "--defaults-torrc", "/dev/null", "--RunAsDaemon", "0", "--User", platform.Detect().TorUser, "--DataDirectory", "/var/lib/shadow21-proxy", "--PidFile", "/run/shadow21-proxy/tor.pid", "--ControlPort", "unix:/run/shadow21-proxy/control", "--ControlSocket", "0", "--CookieAuthentication", "1", "--CookieAuthFile", "/run/shadow21-proxy/control.authcookie", "--CookieAuthFileGroupReadable", "0", "--SocksPort", Endpoint, "--SocksPolicy", "accept *", "--TransPort", "0", "--DNSPort", "0", "--Log", "notice stdout"}
}
func (m Manager) report(message string) {
	if m.Report != nil {
		m.Report(message)
	}
}
func (m Manager) Start(ctx context.Context, uid int, n string) error {
	m.report("Checking session ownership and routing state…")
	if e := ValidateName(n); e != nil {
		return e
	}
	if e := m.RoutingCheck(); e != nil {
		return e
	}
	if e := os.MkdirAll(m.RuntimeDir, 0700); e != nil {
		return e
	}
	s, e := m.read()
	fresh := os.IsNotExist(e)
	if fresh {
		if _, err := os.Lstat(m.PublicPath); err == nil {
			return fmt.Errorf("unfinished proxy status; recover first")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if e != nil && !fresh {
		return e
	}
	if !fresh {
		if e = m.own(s, uid); e != nil {
			return e
		}
		if s.Phase != "active" {
			return fmt.Errorf("unfinished proxy operation; recover first")
		}
		for _, x := range s.Sessions {
			if x == n {
				return fmt.Errorf("session already exists")
			}
		}
		return fmt.Errorf("one managed tmux session is already active; stop it first")
	}
	if fresh {
		m.report("Checking the local Tor proxy…")
		existing := ""
		if m.Existing != nil {
			existing, e = m.Existing(ctx)
			if e != nil {
				return e
			}
		}
		if e = ValidateExistingUnit(existing); e != nil {
			return e
		}
		m.report("Checking Tor, tmux and system services…")
		if e = m.Preflight(ctx); e != nil {
			return e
		}
		s = State{Owner: uid, Sessions: []string{}, Phase: "starting", ExistingTorUnit: existing}
		if e = m.save(s); e != nil {
			return e
		}
		if existing == "" {
			if e = m.configure(ctx, &s); e != nil {
				return m.rollback(ctx, s, e)
			}
			s.TorStarted = true
			if e = m.save(s); e != nil {
				return m.rollback(ctx, s, e)
			}
			a := []string{"--unit=" + TorUnit, "--collect", "--property=Type=exec", "--property=KillMode=control-group", "--property=TimeoutStopSec=10", "--property=UMask=0077", "--", "tor"}
			a = append(a, torArgs(m.ConfigPath)...)
			if e = m.command(ctx, "systemd-run", a...); e != nil {
				return m.rollback(ctx, s, e)
			}
		}
	}
	if s.ExistingTorUnit == "" {
		m.report("Waiting for Tor bootstrap and SOCKS readiness…")
		if e = m.Ready(ctx); e != nil {
			return m.rollback(ctx, s, e)
		}
	}
	if e = m.BindCheck(ctx); e != nil {
		return m.rollback(ctx, s, e)
	}
	m.report("Creating the owned tmux session…")
	s.Sessions = append(s.Sessions, n)
	s.Phase = "starting"
	if e = m.save(s); e != nil {
		return m.rollback(ctx, s, e)
	}
	a := []string{"--unit=" + unit(n), "--collect", "--property=Type=exec", "--property=KillMode=control-group", "--property=TimeoutStopSec=5", "--property=UMask=0077", "--setenv=ALL_PROXY=socks5h://" + Endpoint, "--setenv=all_proxy=socks5h://" + Endpoint, "--setenv=HOME=/root", "--setenv=HISTFILE=/dev/null", "--", "tmux", "-D", "-f", "/dev/null", "-S", m.socket(n)}
	if e = m.command(ctx, "systemd-run", a...); e != nil {
		return m.rollback(ctx, s, e)
	}
	if e = m.waitServer(ctx, n); e != nil {
		return m.rollback(ctx, s, e)
	}
	args := []string{"new-session", "-d"}
	if n != "0" {
		args = append(args, "-s", n)
	}
	args = append(args, "-e", "ALL_PROXY=socks5h://"+Endpoint, "-e", "all_proxy=socks5h://"+Endpoint, "-e", "HISTFILE=/dev/null", rootShell()+" -l")
	if e = m.tmux(ctx, n, args...); e != nil {
		return m.rollback(ctx, s, e)
	}
	if e = m.tmux(ctx, n, "has-session", "-t", "="+n); e != nil {
		return m.rollback(ctx, s, e)
	}
	// -D keeps the initially empty server alive. Once a session exists, let
	// its service exit when the last pane closes instead of reporting an empty
	// server as a live session. Detached sessions still keep the server alive.
	if e = m.tmux(ctx, n, "set-option", "-g", "exit-empty", "on"); e != nil {
		return m.rollback(ctx, s, e)
	}
	s.Phase = "active"
	if e = m.save(s); e != nil {
		return m.rollback(ctx, s, e)
	}
	return nil
}
func (m Manager) waitServer(ctx context.Context, n string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		if m.tmux(ctx, n, "show-options", "-g", "exit-empty") == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("tmux server did not become ready")
		case <-ticker.C:
		}
	}
}
func (m Manager) rollback(ctx context.Context, s State, cause error) error {
	// Readiness cancellation must not cancel the recovery operations as well.
	recovery, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	if e := m.cleanup(recovery, &s); e != nil {
		s.Phase = "failed"
		_ = m.save(s)
		return errors.Join(cause, fmt.Errorf("rollback failed; recovery required: %w", e))
	}
	return cause
}
func (m Manager) stopSession(ctx context.Context, n string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	// kill-session is scoped to the session's private socket. The service stop
	// then kills surviving descendants still in its cgroup, including nohup jobs.
	_ = m.tmux(ctx, n, "kill-session", "-t", "="+n)
	active, e := m.active(ctx, unit(n))
	if e != nil {
		return e
	}
	if active {
		if e = m.command(ctx, "systemctl", "stop", unit(n)); e != nil {
			return e
		}
	}
	active, e = m.active(ctx, unit(n))
	if e != nil {
		return e
	}
	if active {
		return fmt.Errorf("session service is still active")
	}
	if _, e := os.Lstat(m.socket(n)); e == nil {
		if m.tmux(ctx, n, "has-session", "-t", "="+n) == nil {
			return fmt.Errorf("session is still present")
		}
		if e = os.Remove(m.socket(n)); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return nil
}
func (m Manager) restore(s *State) error {
	if !s.ConfigChanged {
		return nil
	}
	if info, e := os.Lstat(m.ConfigPath); e == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("Tor configuration type changed externally")
	}
	current, e := os.ReadFile(m.ConfigPath)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	h := hash(current)
	if h != s.ManagedHash && !(s.OriginalExists && h == s.OriginalHash) && !(!s.OriginalExists && os.IsNotExist(e)) {
		return fmt.Errorf("Tor configuration changed externally; preserve it and resolve recovery manually")
	}
	backup, e := os.ReadFile(m.backupPath())
	if e != nil {
		return e
	}
	if hash(backup) != s.OriginalHash {
		return fmt.Errorf("Tor backup checksum mismatch")
	}
	if s.OriginalExists {
		if e = fileops.WriteAtomic(m.ConfigPath, backup, os.FileMode(s.OriginalMode)); e != nil {
			return e
		}
		if s.OriginalOwnerRecorded {
			if e = os.Chown(m.ConfigPath, s.OriginalUID, s.OriginalGID); e != nil {
				return e
			}
		}
	} else if e = fileops.RemoveIfExists(m.ConfigPath); e != nil {
		return e
	}
	s.ConfigChanged = false
	return m.save(*s)
}
func (m Manager) cleanup(ctx context.Context, s *State) error {
	for _, n := range s.Sessions {
		if e := m.stopSession(ctx, n); e != nil {
			return e
		}
	}
	if s.TorStarted {
		a, e := m.active(ctx, TorUnit)
		if e != nil {
			return e
		}
		if a {
			if e = m.command(ctx, "systemctl", "stop", TorUnit); e != nil {
				return e
			}
		}
		a, e = m.active(ctx, TorUnit)
		if e != nil {
			return e
		}
		if a {
			return fmt.Errorf("managed Tor is still active")
		}
		s.TorStarted = false
		if e = m.save(*s); e != nil {
			return e
		}
	}
	if e := m.restore(s); e != nil {
		return e
	}
	if e := fileops.RemoveIfExists(m.backupPath()); e != nil {
		return e
	}
	if e := fileops.RemoveIfExists(m.statePath()); e != nil {
		return e
	}
	if e := fileops.SyncDirectory(m.StateDir); e != nil {
		return e
	}
	return fileops.RemoveIfExists(m.PublicPath)
}
func (m Manager) Stop(ctx context.Context, uid int, n string) error {
	if e := ValidateName(n); e != nil {
		return e
	}
	s, e := m.read()
	if os.IsNotExist(e) {
		return m.clearCompletedStatus(ctx, uid)
	}
	if e != nil {
		return e
	}
	if e = m.own(s, uid); e != nil {
		return e
	}
	found := false
	for _, x := range s.Sessions {
		if x == n {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("session is not owned by shadow21")
	}
	s.Phase = "stopping"
	if e = m.save(s); e != nil {
		return e
	}
	if e = m.stopSession(ctx, n); e != nil {
		s.Phase = "failed"
		_ = m.save(s)
		return e
	}
	names := []string{}
	for _, x := range s.Sessions {
		if x != n {
			names = append(names, x)
		}
	}
	s.Sessions = names
	if len(names) == 0 {
		if e = m.cleanup(ctx, &s); e != nil {
			s.Phase = "failed"
			_ = m.save(s)
		}
		return e
	}
	s.Phase = "active"
	return m.save(s)
}
func (m Manager) Recover(ctx context.Context, uid int) error {
	s, e := m.read()
	if os.IsNotExist(e) {
		return m.clearCompletedStatus(ctx, uid)
	}
	if e != nil {
		return e
	}
	if e = m.own(s, uid); e != nil {
		return e
	}
	s.Phase = "stopping"
	if e = m.save(s); e != nil {
		return e
	}
	if e = m.cleanup(ctx, &s); e != nil {
		s.Phase = "failed"
		_ = m.save(s)
	}
	return e
}
func (m Manager) Action(ctx context.Context, uid int, op, n string) error {
	if e := ValidateName(n); e != nil {
		return e
	}
	s, e := m.read()
	if e != nil {
		return e
	}
	if e = m.own(s, uid); e != nil {
		return e
	}
	if s.Phase != "active" {
		return fmt.Errorf("proxy mode is not ready")
	}
	found := false
	for _, x := range s.Sessions {
		if x == n {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("session is not owned by shadow21")
	}
	if op == "detach" {
		return m.tmux(ctx, n, "detach-client", "-s", n)
	}
	if op != "attach" {
		return fmt.Errorf("invalid session action")
	}
	c := attachmentCommand(m.socket(n), n)
	return c.Run()
}

func attachmentCommand(socket, name string) *exec.Cmd {
	c := exec.Command("tmux", "-N", "-f", "/dev/null", "-S", socket, "attach-session", "-t", "="+name)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	// pkexec may remove TERM; the embedded client implements this terminal type.
	c.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "HOME=/root", "TERM=xterm-256color", "LANG=C.UTF-8"}
	return c
}
func preflight(ctx context.Context) error {
	d := platform.Detect()
	if d.Family == "" {
		return fmt.Errorf("unsupported distribution")
	}
	if _, e := os.Stat("/run/systemd/system"); e != nil {
		return fmt.Errorf("systemd is required for managed proxy sessions")
	}
	for _, p := range []string{"tor", "tmux", "systemctl", "systemd-run", "ss"} {
		if _, e := exec.LookPath(p); e != nil {
			return fmt.Errorf("missing dependency: %s", p)
		}
	}
	version, e := run(ctx, "tmux", "-V")
	if e != nil {
		return fmt.Errorf("tmux version unavailable")
	}
	if e = ValidateTmuxVersion(string(version)); e != nil {
		return e
	}
	for _, u := range []string{"anonsurfd.service", "wvpn21d.service", "shadow21d.service"} {
		if d.Family == "debian" && u == "tor.service" {
			b, e := run(ctx, "systemctl", "show", u, "--property=SubState", "--value")
			if e == nil && strings.TrimSpace(string(b)) == "exited" {
				continue
			}
		}
		b, e := run(ctx, "systemctl", "show", u, "--property=ActiveState", "--value")
		if e != nil {
			if missingUnit(string(b)) {
				continue
			}
			return fmt.Errorf("cannot inspect existing services")
		}
		switch strings.TrimSpace(string(b)) {
		case "inactive", "failed":
		default:
			return fmt.Errorf("existing Tor or routing service must be stopped explicitly first")
		}
	}
	u, e := user.Lookup(d.TorUser)
	if e != nil {
		return fmt.Errorf("Tor service account unavailable")
	}
	uid, e := strconv.Atoi(u.Uid)
	if e != nil {
		return e
	}
	gid, e := strconv.Atoi(u.Gid)
	if e != nil {
		return e
	}
	b, e := run(ctx, "ss", "-H", "-ltn", "sport = :9050")
	if e != nil {
		return fmt.Errorf("SOCKS port cannot be inspected")
	}
	if len(strings.TrimSpace(string(b))) != 0 {
		if e = ParseBindings(string(b)); e != nil {
			return e
		}
	}
	for _, p := range []string{"/run/shadow21-proxy", "/var/lib/shadow21-proxy"} {
		if e = os.MkdirAll(p, 0700); e != nil {
			return e
		}
		if e = os.Chown(p, uid, gid); e != nil {
			return e
		}
		if e = os.Chmod(p, 0700); e != nil {
			return e
		}
	}
	return nil
}
func bindings(ctx context.Context) error {
	b, e := run(ctx, "ss", "-H", "-ltn", "sport = :9050")
	if e != nil {
		return e
	}
	return ParseBindings(string(b))
}
func ParseBindings(output string) error {
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			return fmt.Errorf("cannot verify SOCKS binding")
		}
		if f[3] != Endpoint {
			return fmt.Errorf("SOCKS listener is not loopback-only")
		}
		count++
	}
	if count != 1 {
		return fmt.Errorf("unexpected SOCKS listeners")
	}
	return nil
}
func ready(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		n, e := tor.BootstrapAt("unix", "/run/shadow21-proxy/control", "/run/shadow21-proxy/control.authcookie")
		if e == nil && n == 100 && netcheck.ProbeSOCKS(ctx, Endpoint) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("managed Tor readiness timed out")
		case <-ticker.C:
		}
	}
}

func missingUnit(output string) bool {
	return strings.Contains(output, "could not be found") || strings.Contains(output, "not loaded")
}

func (m Manager) clearCompletedStatus(ctx context.Context, uid int) error {
	b, e := os.ReadFile(m.PublicPath)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	var p Public
	if json.Unmarshal(b, &p) != nil || p.Mode != "proxy" || p.Owner < 0 {
		return fmt.Errorf("invalid stale proxy status")
	}
	if uid != 0 && p.Owner != uid {
		return fmt.Errorf("proxy mode belongs to another user")
	}
	if _, e = os.Stat(m.backupPath()); !os.IsNotExist(e) {
		return fmt.Errorf("protected proxy journal is missing; manual recovery required")
	}
	units := []string{TorUnit}
	for _, n := range p.Sessions {
		if e = ValidateName(n); e != nil {
			return e
		}
		units = append(units, unit(n))
	}
	for _, u := range units {
		a, e := m.active(ctx, u)
		if e != nil {
			return e
		}
		if a {
			return fmt.Errorf("owned service remains without its protected journal; manual recovery required")
		}
	}
	return fileops.RemoveIfExists(m.PublicPath)
}

func ValidateTmuxVersion(version string) error {
	re := regexp.MustCompile(`^tmux ([0-9]+)\.([0-9]+)`)
	m := re.FindStringSubmatch(strings.TrimSpace(version))
	if len(m) != 3 {
		return fmt.Errorf("unsupported tmux version")
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	if major < 3 || (major == 3 && minor < 2) {
		return fmt.Errorf("tmux 3.2 or newer is required")
	}
	return nil
}
