package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"os"
	"os/exec"
	"strings"
)

type networkSettings struct {
	Bridge     string `json:"bridge"`
	AutoBridge bool   `json:"auto_bridge"`
	SafeSocks  bool   `json:"safe_socks"`
	KillSwitch bool   `json:"kill_switch"`
}

func networkCommand() *cobra.Command {
	root := &cobra.Command{Use: "network", Short: "Read or update authorized system settings"}
	root.AddCommand(&cobra.Command{Use: "read", Short: "Read saved system settings", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		b, e := networkHelper("config-read", nil)
		if e != nil {
			return e
		}
		_, e = c.OutOrStdout().Write(b)
		return e
	}})
	var bridge string
	var auto, safe, kill bool
	apply := &cobra.Command{Use: "apply", Short: "Update settings for the next connection", Args: cobra.NoArgs, RunE: func(c *cobra.Command, a []string) error {
		if len(bridge) > 4096 || strings.ContainsAny(bridge, "\r\n") {
			return fmt.Errorf("invalid bridge line")
		}
		b, e := networkHelper("config-read", nil)
		if e != nil {
			return e
		}
		var cfg networkSettings
		if e = json.Unmarshal(b, &cfg); e != nil {
			return e
		}
		if c.Flags().Changed("bridge") {
			cfg.Bridge = bridge
		}
		if c.Flags().Changed("auto-bridge") {
			cfg.AutoBridge = auto
		}
		if c.Flags().Changed("safe-socks") {
			cfg.SafeSocks = safe
		}
		if c.Flags().Changed("kill-switch") {
			cfg.KillSwitch = kill
		}
		data, e := json.Marshal(cfg)
		if e != nil {
			return e
		}
		b, e = networkHelper("config", data)
		if e != nil {
			return e
		}
		_, e = c.OutOrStdout().Write(b)
		return e
	}}
	apply.Flags().StringVar(&bridge, "bridge", "", "Single bridge line; empty disables manual bridges")
	apply.Flags().BoolVar(&auto, "auto-bridge", false, "Use the installed bridge list")
	apply.Flags().BoolVar(&safe, "safe-socks", true, "Enable Tor SafeSocks")
	apply.Flags().BoolVar(&kill, "kill-switch", true, "Reject non-exempt direct public traffic")
	root.AddCommand(apply)
	return root
}
func networkHelper(action string, input []byte) ([]byte, error) {
	executable, args := command.PrivilegedCommand("/usr/lib/shadow21/shadow21-helper", action)
	cmd := exec.Command(executable, args...)
	cmd.Stdin = os.Stdin
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	cmd.Stderr = os.Stderr
	return cmd.Output()
}
