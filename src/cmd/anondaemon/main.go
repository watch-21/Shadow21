package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/installconfig"
	"gitlab.com/parrotsec/packages/anonsurf/internal/macaddr"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.com/parrotsec/packages/anonsurf/internal/anonsurf"
	"gitlab.com/parrotsec/packages/anonsurf/internal/monitor"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_ = systemd.NotifyStatus("Failed: " + err.Error())
		if len(os.Args) > 1 && (strings.HasPrefix(os.Args[1], "mac-") || os.Args[1] == "prepare-tor") {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "operation", "detail": err.Error()})
		}
		if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "session-") {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"error": sessionErrorCode(err), "detail": boundedDetail(err)})
		}
		fmt.Fprintln(os.Stderr, "shadow21-helper:", err)
		if len(os.Args) > 1 && (os.Args[1] == "session-console" || os.Args[1] == "session-attach") {
			if info, e := os.Stdin.Stat(); e == nil && info.Mode()&os.ModeCharDevice != 0 {
				text := "Press Enter to close this terminal."
				if strings.HasPrefix(os.Getenv("LANG"), "de") {
					text = "Zum Schließen dieses Terminals Eingabetaste drücken."
				}
				fmt.Fprintln(os.Stderr, text)
				var line string
				fmt.Fscanln(os.Stdin, &line)
			}
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("expected one command: start, stop, restart, recover, changeid, or monitor")
	}

	if os.Geteuid() != 0 {
		return fmt.Errorf("privileged helper requires root")
	}
	os.Setenv("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
	if len(args) == 2 && args[0] == "session-console" {
		return consoleSession(args[1])
	}
	if len(args) == 2 && args[0] == "session-attach" {
		return proxymode.New().Action(context.Background(), callingUID(), "attach", args[1])
	}
	if len(args) == 2 && !strings.HasPrefix(args[0], "session-") && args[0] != "mac-random" && args[0] != "mac-restore" {
		return fmt.Errorf("unexpected command argument")
	}
	if len(args) == 1 && args[0] == "prepare-tor" {
		return prepareTorSelection()
	}
	if args[0] == "monitor" {
		return monitor.NewNyxMonitor(os.Stdin, os.Stdout, os.Stderr).Run()
	}
	if err := os.MkdirAll("/run/shadow21", 0700); err != nil {
		return err
	}
	release, err := modes.Lock("/run/shadow21/lifecycle.lock")
	if err != nil {
		return err
	}
	defer release()

	os.Setenv("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
	controller := anonsurf.NewController(newStatusReporter())
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	proxy := proxymode.New()
	switch args[0] {

	case "mac-random", "mac-restore":
		if len(args) != 2 {
			return fmt.Errorf("expected a network interface")
		}
		if args[0] == "mac-random" {
			return macaddr.New().Apply(ctx, callingUID(), args[1])
		}
		return macaddr.New().Restore(ctx, callingUID(), args[1])
	case "mac-recover":
		return macaddr.New().RestoreAll(ctx, callingUID())
	case "install-proxy-config", "uninstall-proxy-config":
		if e := modes.RequireSystem(); e != nil {
			return e
		}
		if e := modes.RequireProxy(); e != nil {
			return e
		}
		manager := installconfig.New()
		if args[0] == "install-proxy-config" {
			return manager.Apply(ctx)
		}
		return manager.Restore()
	case "session-start", "session-stop", "session-detach":
		if len(args) != 2 {
			return fmt.Errorf("expected a session name")
		}
		switch args[0] {
		case "session-start":
			if e := macaddr.New().RequireReady(); e != nil {
				return e
			}
			return proxy.Start(ctx, callingUID(), args[1])
		case "session-stop":
			return proxy.Stop(ctx, callingUID(), args[1])
		default:
			return proxy.Action(ctx, callingUID(), "detach", args[1])
		}
	case "session-recover":
		return proxy.Recover(ctx, callingUID())
	case "config-read":
		return readSettings()
	case "config":
		return applySettings()
	case "start":
		if e := macaddr.New().RequireReady(); e != nil {
			return e
		}
		return controller.Start()
	case "stop":
		return controller.Stop()
	case "restart":
		return controller.Restart()
	case "boot-recover":
		if _, e := os.Stat(proxymode.New().PublicPath); os.IsNotExist(e) {
			if e = proxy.Recover(ctx, 0); e != nil {
				return e
			}
		}
		if e := controller.Recover(); e != nil {
			return e
		}
		return macaddr.New().RecoverInterrupted(ctx)
	case "recover":
		if err := proxy.Recover(ctx, 0); err != nil {
			return err
		}
		if e := controller.Recover(); e != nil {
			return e
		}
		return macaddr.New().RestoreAll(ctx, 0)
	case "changeid":
		return controller.ChangeIdentity()
	case "monitor":
		return monitor.NewNyxMonitor(os.Stdin, os.Stdout, os.Stderr).Run()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func newStatusReporter() anonsurf.StatusReporter {
	// Notification failure path:
	//   publish phase -> socket error -> log once -> disable later reports
	// Reporting is observability only. Firewall, DNS and Tor transitions keep
	// running because failing them would turn a UI problem into a network or
	// recovery failure.
	available := true
	return func(status string) {
		if !available {
			return
		}
		if err := systemd.NotifyStatus(status); err != nil {
			fmt.Fprintln(os.Stderr, "shadow21-helper: report service progress:", err)
			available = false
		}
	}
}

func callingUID() int {
	for _, k := range []string{"PKEXEC_UID", "SUDO_UID"} {
		if v := os.Getenv(k); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n >= 0 {
				return n
			}
		}
	}
	return 0
}

func boundedDetail(err error) string {
	text := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, err.Error())
	if len(text) > 1000 {
		text = text[:1000]
	}
	return text
}
func sessionErrorCode(err error) string {
	var readiness *proxymode.ProxyReadinessError
	if errors.As(err, &readiness) {
		return "proxy-readiness"
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "existing Tor or routing service must be stopped"):
		return "tor-conflict"
	case strings.Contains(text, "SOCKS port") || strings.Contains(text, "not loopback-only"):
		return "port-conflict"
	case strings.Contains(text, "changed externally"):
		return "config-conflict"
	case strings.Contains(text, "conflicting mode"):
		return "mode-conflict"
	case strings.Contains(text, "another lifecycle"):
		return "busy"
	case strings.Contains(text, "missing dependency"):
		return "dependency"
	case strings.Contains(text, "belongs to another"):
		return "owner"
	case strings.Contains(text, "readiness timed out"):
		return "bootstrap"
	case strings.Contains(text, "advanced Tor") || strings.Contains(text, "manual migration") || strings.Contains(text, "unsupported"):
		return "unsupported"
	default:
		return "operation"
	}
}
