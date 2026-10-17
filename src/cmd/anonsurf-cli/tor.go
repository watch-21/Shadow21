package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/anonsurf"
	"gitlab.com/parrotsec/packages/anonsurf/internal/cli"
	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
)

func changeIDCommand(changer anonsurf.IdentityClient) *cobra.Command {
	return &cobra.Command{
		Use:   "changeid",
		Short: "Request a new Tor identity",
		Long: "Request new Tor circuits for new connections through Tor's control port.\n" +
			"A different exit IP is not guaranteed; existing connections are unchanged.",
		Args: cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			printer.Running("Requesting a new Tor identity...")
			if err := changer.Change(); err != nil {
				return fmt.Errorf("changeid: %w", err)
			}
			printer.Success("Tor identity change requested.")
			return nil
		},
	}
}

func ipCommand(checker netcheck.Checker) *cobra.Command {
	var useJSON bool
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Verify the public IP and Tor",
		Long: "Check the public IP address used by this request and whether it is seen as Tor traffic.\n" +
			"The result does not verify every application's traffic.",
		Args: cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := console.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
			printer.Running("Checking Tor connectivity...")
			result, err := checker.Check(cmd.Context())
			if err != nil {
				return fmt.Errorf("ip: %w", err)
			}
			if useJSON {
				if err := writeJSON(cmd.OutOrStdout(), newIPOutput(result)); err != nil {
					return fmt.Errorf("write IP JSON: %w", err)
				}
				return nil
			}
			printer.Resultf("Public IP: %s\n", result.IP)
			if result.IsTor {
				printer.Result("Tor:       yes")
			} else {
				printer.Result("Tor:       no")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&useJSON, "json", false, "Print the Tor check result as JSON")
	return cmd
}

type ipOutput struct {
	IP        string `json:"ip"`
	ElapsedMS int64  `json:"elapsed_ms"`
	IsTor     bool   `json:"is_tor"`
	Timestamp string `json:"timestamp"`
	Scope     string `json:"scope"`
}

func newIPOutput(result netcheck.Result) ipOutput {
	return ipOutput{IP: result.IP, IsTor: result.IsTor, ElapsedMS: result.ElapsedMS, Timestamp: time.Now().UTC().Format(time.RFC3339), Scope: "system"}
}
