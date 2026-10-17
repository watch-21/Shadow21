package main

import (
	"context"
	"errors"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/i18n"
	"gitlab.com/parrotsec/packages/anonsurf/internal/macaddr"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"os"
	"strings"
	"time"
)

// Preparation and attachment share one authorization in the external terminal.
// The lifecycle lock must be released before the long-lived interactive client
// starts so that Stop remains available from the interface and CLI.
func consoleSession(name string) error {
	if e := proxymode.ValidateName(name); e != nil {
		return e
	}
	if e := os.MkdirAll("/run/shadow21", 0700); e != nil {
		return e
	}
	release, e := modes.Lock("/run/shadow21/lifecycle.lock")
	if e != nil {
		return e
	}
	if strings.HasPrefix(os.Getenv("LANG"), "de") {
		i18n.Set("de")
	}
	manager := proxymode.New()
	manager.Report = func(message string) { fmt.Fprintln(os.Stderr, i18n.T(message)) }
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	e = macaddr.New().RequireReady()
	if e == nil {
		e = manager.Start(ctx, callingUID(), name)
	}
	cancel()
	release()
	if e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, i18n.T("Tor is ready. Attaching the root tmux session…"))
	if e = manager.Action(context.Background(), callingUID(), "attach", name); e == nil {
		return nil
	}
	recovery, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()
	release, lockError := modes.Lock("/run/shadow21/lifecycle.lock")
	if lockError != nil {
		return errors.Join(e, fmt.Errorf("attachment failed; stop or recover the owned session"))
	}
	defer release()
	return errors.Join(e, manager.Stop(recovery, callingUID(), name))
}
