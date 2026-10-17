package main

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/cli"
	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
	"gitlab.com/parrotsec/packages/anonsurf/internal/dns"
)

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	manager := dns.Manager{}
	var verbose bool
	rootCmd := &cobra.Command{
		Use:   "shadow21-dnstool",
		Short: "Manage system DNS settings",
		Long: "shadow21-dnstool manages the system DNS settings used by shadow21.\n" +
			"Commands that write " + dns.ResolvConfPath + " require root privileges.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cli.Subcommands("command"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Help(); err != nil {
				return err
			}
			return showDNSStatus(console.New(cmd.OutOrStdout(), cmd.ErrOrStderr()))
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				command.SetVerboseOutput(stderr)
				return
			}
			command.SetVerboseOutput(nil)
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetFlagErrorFunc(cli.FlagError)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show commands and their output")
	rootCmd.AddCommand(
		statusCommand(),
		createBackupCommand(manager),
		restoreBackupCommand(manager),
		addressCommand(manager),
	)
	for _, child := range rootCmd.Commands() {
		if child.Name() == "status" {
			continue
		}
		previous := child.RunE
		child.RunE = func(c *cobra.Command, a []string) error {
			if e := requireAdmin(); e != nil {
				return e
			}
			release, e := modes.Lock("/run/shadow21/lifecycle.lock")
			if e != nil {
				return e
			}
			defer release()
			if e = modes.RequireSystem(); e != nil {
				return e
			}
			return previous(c, a)
		}
	}
	return rootCmd
}

func statusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current DNS status",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showDNSStatus(console.New(cmd.OutOrStdout(), cmd.ErrOrStderr()))
		},
	}
}

func createBackupCommand(manager dns.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "create-backup",
		Short: "Create a backup of resolv.conf",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAdmin(); err != nil {
				return err
			}
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			printer.Running("Creating the DNS backup...")
			if err := manager.MakeBackup(); err != nil {
				return fmt.Errorf("create DNS backup: %w", err)
			}
			printer.Success("DNS backup created.")
			return nil
		},
	}
}

func restoreBackupCommand(manager dns.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "restore-backup",
		Short: "Restore resolv.conf from backup",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAdmin(); err != nil {
				return err
			}
			return restoreDNS(manager, console.New(cmd.OutOrStdout(), cmd.ErrOrStderr()))
		},
	}
}

func addressCommand(manager dns.Manager) *cobra.Command {
	return &cobra.Command{
		Use:     "address [dhcp] [addr...]",
		Aliases: []string{"addr"},
		Short:   "Write custom DNS addresses or use DHCP",
		RunE: func(cmd *cobra.Command, args []string) error {
			result := dns.ParseAddrFromParams(args)
			if len(result.Invalid) > 0 {
				return cli.NewError(
					fmt.Sprintf("invalid DNS address: %s", strings.Join(result.Invalid, ", ")),
					"Use an IPv4 or IPv6 address, such as 9.9.9.9 or 2620:fe::fe, or run 'sudo shadow21-dnstool address dhcp'.",
				)
			}
			if err := requireAdmin(); err != nil {
				return err
			}
			return writeDNSAddresses(
				manager,
				console.New(cmd.OutOrStdout(), cmd.ErrOrStderr()),
				result.HasDHCPFlag,
				result.ListAddr,
			)
		},
	}
}

func requireAdmin() error {
	if !command.IsAdmin() {
		return errAdminRequired
	}
	return nil
}
