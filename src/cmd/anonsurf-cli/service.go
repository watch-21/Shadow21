package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/cli"
	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
	"gitlab.com/parrotsec/packages/anonsurf/internal/safekill"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

func serviceCommand(use, short, activity string, run func() error, success string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			printer.Running(activity)
			if err := run(); err != nil {
				return fmt.Errorf("%s: %w", use, err)
			}
			printer.Success(success)
			return nil
		},
	}
}

func serviceCommandWithSafeKill(
	use, short, activity string,
	run func() error,
	readStatus func() (string, error),
	success string,
	safeKill safekill.Runner,
) *cobra.Command {
	var useSafeKill bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			if useSafeKill {
				printer.Running("Closing networked applications and clearing caches...")
				if err := safeKill.Run(); err != nil {
					return fmt.Errorf("%s safe-kill: %w", use, err)
				}
				printer.Success("SafeKill completed.")
			}
			printer.Running(activity)
			if err := runWithServiceProgress(printer, run, readStatus); err != nil {
				return fmt.Errorf("%s: %w", use, err)
			}
			printer.Success(success)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&useSafeKill, "safe-kill", "k", false, "Run safekill before changing shadow21 service state")
	return cmd
}

func runWithServiceProgress(
	printer console.Printer,
	run func() error,
	readStatus func() (string, error),
) error {
	// Live progress is coordinated as follows:
	//
	//   1. Read persisted StatusText as the baseline.
	//   2. Run blocking systemctl in a worker goroutine.
	//   3. Poll StatusText here and print only changed values.
	//   4. Return the worker result when the transition finishes.
	//
	// The baseline matters because RemainAfterExit keeps the previous final
	// StatusText: without it, stale text would look like progress from this run.
	lastStatus, err := readStatus()
	if err != nil {
		printer.Warning(fmt.Sprintf("Live service progress is unavailable: %v", err))
		return run()
	}

	result := make(chan error, 1)
	go func() {
		result <- run()
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	progressAvailable := true
	for {
		select {
		case err := <-result:
			return err
		case <-ticker.C:
			if !progressAvailable {
				continue
			}
			statusText, err := readStatus()
			if err != nil {
				printer.Warning(fmt.Sprintf("Live service progress stopped: %v", err))
				progressAvailable = false
				continue
			}
			if statusText != "" && statusText != lastStatus {
				printer.Running(statusText)
				lastStatus = statusText
			}
		}
	}
}

func bootCommand(service systemd.ServiceController) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boot",
		Short: "Manage startup integration",
		Args:  cli.Subcommands("action"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		serviceCommand(
			"enable", "Enable shadow21 at system startup", "Enabling shadow21 at startup...",
			service.Enable, "shadow21 enabled at startup.",
		),
		serviceCommand(
			"disable", "Disable shadow21 at system startup", "Disabling shadow21 at startup...",
			service.Disable, "shadow21 disabled at startup.",
		),
		statusBootCommand(service),
	)
	return cmd
}

func statusBootCommand(service systemd.ServiceController) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether shadow21 starts at boot",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			enabled, err := service.IsEnabled()
			if err != nil {
				return fmt.Errorf("boot status: %w", err)
			}
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			if enabled {
				printer.Result("shadow21 is enabled at startup.")
			} else {
				printer.Result("shadow21 is disabled at startup.")
			}
			return nil
		},
	}
}
