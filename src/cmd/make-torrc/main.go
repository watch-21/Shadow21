package main

import (
	"errors"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "make-torrc:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges are required")
	}
	release, e := modes.Lock("/run/shadow21/lifecycle.lock")
	if e != nil {
		return e
	}
	defer release()
	if e = modes.RequireSystem(); e != nil {
		return e
	}
	if len(args) == 0 {
		return generateTorrc()
	}
	if len(args) > 1 {
		return fmt.Errorf("expected at most one argument")
	}

	switch args[0] {
	case "new-torrc":
		return generateTorrc()
	case "restore":
		if err := (tor.TorrcManager{}).Restore(); err != nil && !errors.Is(err, tor.ErrTorrcBackupNotFound) {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown argument %q", args[0])
	}
}

func generateTorrc() error {
	return (tor.Configurator{}).Generate()
}
