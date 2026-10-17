package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
)

const Socket = "shadow21"
const Helper = "/usr/lib/shadow21/shadow21-helper"

func ValidateName(n string) error { return proxymode.ValidateName(n) }

type Manager struct {
	Settings   preferences.Settings
	UID        int
	PublicPath string
	Execute    func(context.Context, string, ...string) ([]byte, error)
}

func New(s preferences.Settings) Manager {
	return Manager{s, os.Getuid(), modes.PublicPath, func(ctx context.Context, p string, a ...string) ([]byte, error) {
		c := exec.CommandContext(ctx, p, a...)
		c.Stdin = os.Stdin
		c.Stderr = os.Stderr
		return c.Output()
	}}
}
func ReadPublic(path string) (proxymode.Public, error) {
	var p proxymode.Public
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return proxymode.Public{Phase: "inactive", Mode: "none", Sessions: []string{}}, nil
	}
	if e != nil {
		return p, e
	}
	if e = json.Unmarshal(b, &p); e != nil {
		return p, e
	}
	if e = proxymode.ValidateExistingUnit(p.ExistingTorUnit); e != nil {
		return p, e
	}
	if p.Mode != "proxy" || p.Owner < 0 {
		return p, fmt.Errorf("invalid proxy status")
	}
	for _, n := range p.Sessions {
		if e = ValidateName(n); e != nil {
			return p, e
		}
	}
	return p, nil
}
func (m Manager) List(ctx context.Context) ([]string, error) {
	p, e := ReadPublic(m.PublicPath)
	if e != nil {
		return nil, e
	}
	if p.Owner != m.UID && m.UID != 0 {
		return []string{}, nil
	}
	return p.Sessions, nil
}
func (m Manager) helper(ctx context.Context, action, n string) error {
	p, a := command.PrivilegedCommand(Helper, action, n)
	_, e := m.Execute(ctx, p, a...)
	if e != nil {
		return fmt.Errorf("authorized session operation failed: %w", e)
	}
	return nil
}
func (m Manager) Start(ctx context.Context, n string) error {
	if !m.Settings.TmuxEnabled {
		return fmt.Errorf("tmux is disabled")
	}
	if m.Settings.Mode != "proxy" {
		return fmt.Errorf("select proxy mode before starting tmux")
	}
	if m.Settings.Endpoint != proxymode.Endpoint {
		return fmt.Errorf("managed tmux requires 127.0.0.1:9050")
	}
	if e := ValidateName(n); e != nil {
		return e
	}
	return m.helper(ctx, "session-start", n)
}
func (m Manager) Action(ctx context.Context, a, n string) error {
	if e := ValidateName(n); e != nil {
		return e
	}
	switch a {
	case "kill-session":
		return m.helper(ctx, "session-stop", n)
	case "detach-client":
		return m.helper(ctx, "session-detach", n)
	default:
		return fmt.Errorf("invalid session action")
	}
}
func Attach(n string, stdin io.Reader, stdout, stderr io.Writer) error {
	if e := ValidateName(n); e != nil {
		return e
	}
	p, a := command.PrivilegedCommand(Helper, "session-attach", n)
	c := exec.Command(p, a...)
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

// Dedicated emulator instances exit when the attached client ends. Shared
// terminal processes are never killed to close one application window.
func OpenTerminal(args ...string) error {
	candidates := []struct {
		name   string
		prefix []string
	}{{"xfce4-terminal", []string{"--disable-server", "--geometry=120x36", "-x"}}, {"qterminal", []string{"-e"}}, {"konsole", []string{"--separate", "--nofork", "--geometry", "1100x740", "-e"}}, {"xterm", []string{"-geometry", "120x36", "-xrm", "XTerm*VT100.translations: #override\nCtrl Shift<Key>C: copy-selection(CLIPBOARD)\nCtrl Shift<Key>V: insert-selection(CLIPBOARD)", "-e"}}}
	if preferred, e := filepath.EvalSymlinks("/usr/bin/x-terminal-emulator"); e == nil {
		for i, c := range candidates {
			if filepath.Base(preferred) == c.name {
				candidates[0], candidates[i] = candidates[i], candidates[0]
				break
			}
		}
	}
	for _, candidate := range candidates {
		p := filepath.Join("/usr/bin", candidate.name)
		info, e := os.Stat(p)
		if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 || info.Mode().Perm()&0022 != 0 {
			continue
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); !ok || stat.Uid != 0 {
			continue
		}
		c := exec.Command(p, append(candidate.prefix, args...)...)
		if e = c.Start(); e != nil {
			continue
		}
		go func() { _ = c.Wait() }()
		return nil
	}
	return fmt.Errorf("no dedicated terminal emulator found; install xterm, Konsole or xfce4-terminal")
}
func Timeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 145*time.Second)
}
func SelectionAllowed(old, next preferences.Settings) error {
	if old.Mode == next.Mode && old.TmuxEnabled == next.TmuxEnabled && old.Endpoint == next.Endpoint {
		return nil
	}
	p, e := ReadPublic(modes.PublicPath)
	if e != nil {
		return e
	}
	if p.Phase != "inactive" {
		return fmt.Errorf("stop or recover proxy mode before changing connection settings")
	}
	if _, e := os.Stat("/run/systemd/system"); os.IsNotExist(e) {
		return nil
	}
	c, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	out, e := exec.CommandContext(c, "systemctl", "show", "shadow21d.service", "--property=ActiveState", "--value").Output()
	// No service manager in a build container: preference editing is harmless;
	// all actual starts still require privileged preflight and the lifecycle lock.
	if e != nil {
		return fmt.Errorf("cannot establish routing service state before changing connection settings")
	}
	{
		state := strings.TrimSpace(string(out))
		if state != "inactive" && state != "failed" {
			return fmt.Errorf("stop system routing before changing connection settings")
		}
	}
	return nil
}

func Observe(ctx context.Context, uid int) (map[string]any, error) {
	p, e := ReadPublic(modes.PublicPath)
	if e != nil {
		return nil, e
	}
	r := map[string]any{"mode": p.Mode, "phase": p.Phase, "owner": p.Owner, "sessions": p.Sessions}
	if p.Phase == "inactive" {
		return r, nil
	}
	if p.Phase == "active" {
		torUnit := proxymode.TorUnit
		if p.ExistingTorUnit != "" {
			torUnit = p.ExistingTorUnit
		}
		units := []string{torUnit}
		for _, n := range p.Sessions {
			units = append(units, "shadow21-tmux-"+n+".service")
		}
		for _, u := range units {
			b, e := exec.CommandContext(ctx, "systemctl", "show", u, "--property=ActiveState", "--value").Output()
			if e != nil {
				r["phase"] = "unknown"
				break
			}
			state := strings.TrimSpace(string(b))
			if state != "active" {
				r["phase"] = "failed"
				break
			}
		}
	}
	if p.Owner != uid && uid != 0 {
		r["sessions"] = []string{}
	}
	return r, nil
}
