package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/diagnostics"
	"gitlab.com/parrotsec/packages/anonsurf/internal/macaddr"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"gitlab.com/parrotsec/packages/anonsurf/internal/sessions"
	"os"
	"os/exec"
	"time"
)

func extras() []*cobra.Command {
	prefs := &cobra.Command{Use: "preferences", Short: "Show or update user preferences", Args: cobra.NoArgs}
	prefs.RunE = func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		return writeJSON(c.OutOrStdout(), s)
	}
	set := &cobra.Command{Use: "set", Short: "Save validated user preferences", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		previous := s
		if c.Flags().Changed("language") {
			s.Language, _ = c.Flags().GetString("language")
		}
		if c.Flags().Changed("mode") {
			s.Mode, _ = c.Flags().GetString("mode")
		}
		if c.Flags().Changed("endpoint") {
			s.Endpoint, _ = c.Flags().GetString("endpoint")
		}
		if c.Flags().Changed("auto-ip") {
			s.AutoIP, _ = c.Flags().GetBool("auto-ip")
		}
		if c.Flags().Changed("tmux") {
			s.TmuxEnabled, _ = c.Flags().GetBool("tmux")
		}
		if e = sessions.SelectionAllowed(previous, s); e != nil {
			return e
		}
		if e = preferences.Save(s); e != nil {
			return e
		}
		return writeJSON(c.OutOrStdout(), s)
	}}
	set.Flags().String("language", "en", "English or German: en/de")
	set.Flags().String("mode", "system", "system or proxy")
	set.Flags().String("endpoint", "127.0.0.1:9050", "Local SOCKS endpoint")
	set.Flags().Bool("auto-ip", true, "Automatically refresh the displayed external IP")
	set.Flags().Bool("tmux", false, "Enable optional tmux sessions")
	prefs.AddCommand(set)
	tm := &cobra.Command{Use: "tmux", Short: "Manage optional authorized root tmux sessions"}
	tm.AddCommand(&cobra.Command{Use: "recover", Short: "Terminate owned proxy mode and restore its configuration", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		b, e := networkHelper("session-recover", nil)
		if e != nil {
			return e
		}
		_, e = c.OutOrStdout().Write(b)
		return e
	}})
	tm.AddCommand(&cobra.Command{Use: "status", Short: "Observe managed proxy mode and service state", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()
		r, e := sessions.Observe(ctx, os.Getuid())
		if e != nil {
			return e
		}
		return writeJSON(c.OutOrStdout(), r)
	}})
	tm.AddCommand(&cobra.Command{Use: "list", Short: "List owned sessions", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		ctx, cancel := sessions.Timeout()
		defer cancel()
		n, e := sessions.New(s).List(ctx)
		if e != nil {
			return e
		}
		return writeJSON(c.OutOrStdout(), n)
	}})
	for _, op := range []string{"start", "stop", "detach", "attach", "open"} {
		op := op
		tm.AddCommand(&cobra.Command{Use: op + " NAME", Short: op + " a session", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, a []string) error {
			s, e := preferences.Load()
			if e != nil {
				return e
			}
			m := sessions.New(s)
			ctx, cancel := sessions.Timeout()
			defer cancel()
			switch op {
			case "start":
				if e = m.Start(ctx, a[0]); e != nil {
					return e
				}
				exe, e := os.Executable()
				if e != nil {
					_ = m.Action(ctx, "kill-session", a[0])
					return e
				}
				if e = sessions.OpenTerminal(exe, "tmux", "attach", a[0]); e != nil {
					cleanup := m.Action(ctx, "kill-session", a[0])
					if cleanup != nil {
						return fmt.Errorf("terminal launch failed and cleanup requires recovery: %w", cleanup)
					}
					return e
				}
				return nil
			case "stop":
				return m.Action(ctx, "kill-session", a[0])
			case "detach":
				return m.Action(ctx, "detach-client", a[0])
			case "attach":
				return sessions.Attach(a[0], os.Stdin, c.OutOrStdout(), c.ErrOrStderr())
			case "open":
				if e = sessions.ValidateName(a[0]); e != nil {
					return e
				}
				exe, e := os.Executable()
				if e != nil {
					return e
				}
				return sessions.OpenTerminal(exe, "tmux", "attach", a[0])
			}
			return nil
		}})
	}
	proxy := &cobra.Command{Use: "proxy", Short: "Test the local Tor SOCKS proxy"}
	proxy.AddCommand(&cobra.Command{Use: "test", Short: "Check one request through the SOCKS proxy", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
		defer cancel()
		r, e := (netcheck.Checker{Endpoint: s.Endpoint}).Check(ctx)
		if e != nil {
			return e
		}
		return writeJSON(c.OutOrStdout(), map[string]any{"ip": r.IP, "is_tor": r.IsTor, "timestamp": time.Now().UTC().Format(time.RFC3339), "scope": "this-request-only", "endpoint": s.Endpoint, "elapsed_ms": r.ElapsedMS})
	}})
	health := &cobra.Command{Use: "health", Short: "Inspect local configuration; no external requests", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(c.Context(), 12*time.Second)
		defer cancel()
		return writeJSON(c.OutOrStdout(), diagnostics.Read(ctx, s))
	}}
	diag := &cobra.Command{Use: "diagnostics", Short: "Print a redacted diagnostic report", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		s, e := preferences.Load()
		if e != nil {
			return e
		}
		ctx, cancel := context.WithTimeout(c.Context(), 12*time.Second)
		defer cancel()
		_, e = c.OutOrStdout().Write(diagnostics.Redacted(diagnostics.Read(ctx, s)))
		return e
	}}
	mac := &cobra.Command{Use: "mac", Short: "Change or restore the local interface MAC address"}
	for _, operation := range []string{"random", "restore"} {
		op := operation
		mac.AddCommand(&cobra.Command{Use: op + " INTERFACE", Short: map[string]string{"random": "Generate a random local MAC address", "restore": "Restore the previous MAC address"}[op], Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, a []string) error {
			if !macaddr.ValidInterface(a[0]) {
				return fmt.Errorf("invalid network interface")
			}
			program, args := command.PrivilegedCommand("/usr/lib/shadow21/shadow21-helper", "mac-"+op, a[0])
			process := exec.CommandContext(c.Context(), program, args...)
			process.Stdin = os.Stdin
			process.Stdout = c.OutOrStdout()
			process.Stderr = c.ErrOrStderr()
			return process.Run()
		}})
	}
	mac.AddCommand(&cobra.Command{Use: "recover", Short: "Restore saved MAC operations", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error { _, e := networkHelper("mac-recover", nil); return e }})
	return []*cobra.Command{prefs, tm, proxy, health, diag, mac}
}
