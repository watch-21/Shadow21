package main

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/i18n"
	"gitlab.com/parrotsec/packages/anonsurf/internal/preferences"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gitlab.com/parrotsec/packages/anonsurf/internal/anonsurf"
	"gitlab.com/parrotsec/packages/anonsurf/internal/cli"
	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/console"
	"gitlab.com/parrotsec/packages/anonsurf/internal/netcheck"
	"gitlab.com/parrotsec/packages/anonsurf/internal/safekill"
	"gitlab.com/parrotsec/packages/anonsurf/internal/status"
	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
)

// version is overridden at release time via -ldflags "-X main.version=..." (see Makefile).
var version = "v1"

func main() {
	if err := execute(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if i18n.Language() == "de" {
			fmt.Fprintln(os.Stderr, "shadow21: Vorgang fehlgeschlagen. Abhängigkeiten, Einstellungen und Autorisierung prüfen. Technische Details: --lang en --verbose.")
			os.Exit(1)
		}
		printer := console.New(os.Stdout, os.Stderr)
		printer.Error(fmt.Sprintf("shadow21: %v", err))
		if hint := cli.Hint(err); hint != "" {
			printer.Info(hint)
		}
		os.Exit(1)
	}
}

func execute(args []string, stdout, stderr io.Writer) error {
	settings, _ := preferences.Load()
	language := settings.Language
	for i, arg := range args {
		if strings.HasPrefix(arg, "--lang=") {
			language = strings.TrimPrefix(arg, "--lang=")
		}
		if arg == "--lang" && i+1 < len(args) {
			language = args[i+1]
		}
	}
	if language != "en" && language != "de" {
		return fmt.Errorf("language must be en or de")
	}
	i18n.Set(language)
	rootCmd := newRootCommand(stdout, stderr)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cobra.EnableCommandSorting = false
	var verbose bool
	var language string
	rootCmd := &cobra.Command{
		Use:           "shadow21",
		Short:         "Manage shadow21 system-wide Tor routing",
		Long:          "shadow21 is a Tor routing and privacy management tool for Linux.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cli.Subcommands("command"),
		Example: `  shadow21 start
  shadow21 status
  shadow21 changeid
  shadow21 boot enable`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
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
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetVersionTemplate("shadow21 {{.Version}}\n")
	rootCmd.SetFlagErrorFunc(cli.FlagError)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&language, "lang", i18n.Language(), "Output language: en/de")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Show commands and their output")

	service := systemd.NewAnonsurfdController()
	monitor := anonsurf.MonitorClient{}
	safeKill := safekill.Runner{}
	rootCmd.AddCommand(
		serviceCommandWithSafeKill(
			"start", "Route traffic through Tor", "Starting shadow21...",
			service.Start, service.StatusText, "shadow21 started.", safeKill,
		),
		serviceCommandWithSafeKill(
			"stop", "Restore the original network", "Restoring the original network...",
			service.Stop, service.StatusText, "shadow21 stopped.", safeKill,
		),
		serviceCommandWithSafeKill(
			"restart", "Rebuild Tor routing", "Restarting shadow21...",
			service.Restart, service.StatusText, "shadow21 restarted.", safeKill,
		),
		statusCommand(status.Reader{}, monitor),
		changeIDCommand(anonsurf.IdentityClient{}),
		ipCommand(netcheck.Checker{}),
		monitorCommand(monitor),
		bootCommand(service),
	)
	rootCmd.AddCommand(extras()...)
	rootCmd.AddCommand(networkCommand())
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultHelpFlag()
	rootCmd.InitDefaultVersionFlag()
	localizeCommand(rootCmd)
	configureRootHelp(rootCmd)

	return rootCmd
}

const helpTemplate = `shadow21

{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

Usage:
  {{.UseLine}}
{{if .HasAvailableSubCommands}}
Commands:
{{range .Commands}}{{if or .IsAvailableCommand (eq .Name "help")}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}
{{if .HasAvailableFlags}}Flags:
{{.LocalFlags.FlagUsages | localizeFlags | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
Global Flags:
{{.InheritedFlags.FlagUsages | localizeFlags | trimTrailingWhitespaces}}
{{end}}{{if .Example}}
Examples:
{{.Example}}
{{end}}`

func localizeCommand(cmd *cobra.Command) {
	cmd.InitDefaultHelpFlag()
	if f := cmd.Flags().Lookup("help"); f != nil {
		if i18n.Language() == "de" {
			f.Usage = "Hilfe anzeigen"
		}
	}
	if f := cmd.Flags().Lookup("version"); f != nil {
		if i18n.Language() == "de" {
			f.Usage = "Version anzeigen"
		}
	}
	cmd.Short = i18n.T(cmd.Short)
	cmd.Long = i18n.T(cmd.Long)
	cmd.Flags().VisitAll(func(f *pflag.Flag) { f.Usage = i18n.T(f.Usage) })
	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) { f.Usage = i18n.T(f.Usage) })
	for _, child := range cmd.Commands() {
		localizeCommand(child)
	}
}
