package proxymode

import (
	"context"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Borrow only the installed Tor executable in a known active system service.
// Reused services and their configuration are never reloaded or stopped.
func existingProxy(ctx context.Context) (string, error) {
	b, e := run(ctx, "ss", "-H", "-ltnp", "sport = :9050")
	if e != nil {
		return "", fmt.Errorf("SOCKS port cannot be inspected")
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", nil
	}
	pid, e := listenerPID(string(b))
	if e != nil {
		return "", e
	}
	exe, e := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if e != nil {
		return "", fmt.Errorf("cannot verify existing Tor process")
	}
	tor, e := exec.LookPath("tor")
	if e != nil {
		return "", fmt.Errorf("missing dependency: tor")
	}
	tor, e = filepath.EvalSymlinks(tor)
	if e != nil || tor != exe {
		return "", fmt.Errorf("SOCKS port is occupied by an unverified process")
	}
	for _, unit := range []string{platform.Detect().TorUnit, "tor.service"} {
		if unit == "" {
			continue
		}
		p, pe := run(ctx, "systemctl", "show", unit, "--property=MainPID", "--value")
		a, ae := run(ctx, "systemctl", "show", unit, "--property=ActiveState", "--value")
		if pe != nil || ae != nil || strings.TrimSpace(string(a)) != "active" || strings.TrimSpace(string(p)) != strconv.Itoa(pid) {
			continue
		}
		probeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		e := waitExistingReadiness(probeCtx, (netcheck.Checker{Endpoint: Endpoint}).Check, 2*time.Second)
		cancel()
		if e != nil {
			return "", e
		}
		return unit, nil
	}
	return "", fmt.Errorf("existing Tor listener is not managed by a supported service; manual migration required")
}

func listenerPID(output string) (int, error) {
	if e := ParseBindings(output); e != nil {
		return 0, e
	}
	m := regexp.MustCompile(`users:\(\("tor",pid=([0-9]+),`).FindStringSubmatch(output)
	if len(m) != 2 {
		return 0, fmt.Errorf("SOCKS port owner is not a verified Tor process")
	}
	pid, e := strconv.Atoi(m[1])
	if e != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid Tor listener PID")
	}
	return pid, nil
}

func ValidateExistingUnit(unit string) error {
	if unit != "" && unit != "tor.service" && unit != "tor@default.service" {
		return fmt.Errorf("unsupported existing Tor service")
	}
	return nil
}

// ProxyReadinessError distinguishes an unavailable checked route from a
// conflicting process or service. Its cause remains available to diagnostics.
type ProxyReadinessError struct{ Cause error }

func (e *ProxyReadinessError) Error() string {
	return fmt.Sprintf("Tor proxy connection check failed: %v", e.Cause)
}
func (e *ProxyReadinessError) Unwrap() error { return e.Cause }
func waitExistingReadiness(ctx context.Context, check func(context.Context) (netcheck.Result, error), delay time.Duration) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return &ProxyReadinessError{Cause: readinessCause(ctx.Err(), last)}
		}
		result, err := check(ctx)
		if err == nil {
			if !result.IsTor {
				return &ProxyReadinessError{Cause: fmt.Errorf("connection check did not confirm a Tor exit")}
			}
			return nil
		}
		last = err
		if attempt == 2 {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return &ProxyReadinessError{Cause: readinessCause(ctx.Err(), last)}
		case <-timer.C:
		}
	}
	return &ProxyReadinessError{Cause: last}
}
func readinessCause(cancel, last error) error {
	if last == nil {
		return cancel
	}
	return fmt.Errorf("%w; last connection error: %v", cancel, last)
}
