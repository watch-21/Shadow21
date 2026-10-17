package main

import (
	"encoding/json"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/config"
	"gitlab.com/parrotsec/packages/anonsurf/internal/modes"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
	"io"
	"os"
	"strings"
)

func applySettings() error {
	if e := modes.RequireSystem(); e != nil {
		return e
	}
	if _, err := os.Stat("/run/shadow21/active"); err == nil {
		return fmt.Errorf("disconnect before changing network settings")
	}
	cfg, err := config.Load(config.SystemPath)
	if err != nil {
		return err
	}
	var input struct {
		Bridge     string `json:"bridge"`
		AutoBridge bool   `json:"auto_bridge"`
		SafeSocks  bool   `json:"safe_socks"`
		KillSwitch bool   `json:"kill_switch"`
	}
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 16385))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&input); err != nil {
		return fmt.Errorf("invalid settings input")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return fmt.Errorf("invalid trailing settings")
	}
	if len(input.Bridge) > 4096 || strings.ContainsAny(input.Bridge, "\r\n") {
		return fmt.Errorf("invalid bridge line")
	}
	if input.AutoBridge && input.Bridge != "" {
		return fmt.Errorf("choose either an automatic or manual bridge")
	}
	cfg.BridgeAddress = input.Bridge
	cfg.AutoBridge = input.AutoBridge
	cfg.UseSafeSocks = input.SafeSocks
	cfg.KillSwitch = input.KillSwitch
	options, err := (tor.BridgeResolver{}).Resolve(tor.TorrcOptionsFromConfig(cfg))
	if err != nil {
		return fmt.Errorf("bridge dependencies or configuration invalid")
	}
	rendered, err := tor.RenderTorrc(options)
	if err != nil {
		return fmt.Errorf("invalid Tor settings")
	}
	if err = tor.ValidateRendered(rendered); err != nil {
		return err
	}
	if err = config.Save(cfg); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]bool{"saved": true})
}

func readSettings() error {
	cfg, err := config.Load(config.SystemPath)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"bridge": cfg.BridgeAddress, "auto_bridge": cfg.AutoBridge, "safe_socks": cfg.UseSafeSocks, "kill_switch": cfg.KillSwitch})
}
