package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/cli"
	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
)

var (
	errAdminRequired = errors.New("root privileges are required")
	errDNSUnusable   = errors.New("DNS configuration is unusable")
)

func main() {
	if err := execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		printer := console.New(os.Stdout, os.Stderr)
		printer.Error(fmt.Sprintf("shadow21-dnstool: %v", err))
		if hint := cli.Hint(err); hint != "" {
			printer.Info(hint)
		}
		if errors.Is(err, errAdminRequired) {
			printer.Info("Run this command with sudo.")
		}
		if errors.Is(err, errDNSUnusable) {
			printer.Info("Run 'sudo shadow21-dnstool restore-backup' to restore system DNS.")
		}
		os.Exit(1)
	}
}

func execute(args []string, stdout, stderr io.Writer) error {
	rootCmd := newRootCommand(stdout, stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}
