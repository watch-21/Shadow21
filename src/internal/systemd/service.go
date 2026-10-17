package systemd

import (
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/platform"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
)

const (
	SystemctlPath = "/usr/bin/systemctl"
	AnonsurfdUnit = "shadow21d.service"
	TorUnit       = "tor@default.service"
)

type ServiceController struct {
	unit       string
	privileged bool
}

func NewAnonsurfdController() ServiceController {
	return ServiceController{
		unit:       AnonsurfdUnit,
		privileged: true,
	}
}

func NewTorController() ServiceController {
	return ServiceController{unit: platform.Detect().TorUnit}
}

func IsSystemStopping() bool {
	output, _ := (command.Runner{}).Output(SystemctlPath, "is-system-running")
	return strings.TrimSpace(string(output)) == "stopping"
}

func (c ServiceController) Start() error {
	return c.run("start")
}

func (c ServiceController) Stop() error {
	return c.run("stop")
}

func (c ServiceController) Restart() error {
	return c.run("reload-or-restart")
}

func (c ServiceController) Enable() error {
	return c.run("enable")
}

func (c ServiceController) Disable() error {
	return c.run("disable")
}

func (c ServiceController) IsEnabled() (bool, error) {
	return c.queryState("is-enabled", map[string]bool{
		"enabled":         true,
		"enabled-runtime": true,
		"disabled":        false,
		"masked":          false,
		"masked-runtime":  false,
		"static":          false,
		"indirect":        false,
		"generated":       false,
		"transient":       false,
	})
}

func (c ServiceController) IsActive() (bool, error) {
	return c.queryState("is-active", map[string]bool{
		"active":       true,
		"inactive":     false,
		"failed":       false,
		"activating":   false,
		"deactivating": false,
		"reloading":    false,
	})
}

func (c ServiceController) IsActiveOrActivating() (bool, error) {
	return c.queryState("is-active", map[string]bool{
		"active":       true,
		"activating":   true,
		"reloading":    true,
		"inactive":     false,
		"failed":       false,
		"deactivating": false,
	})
}

func (c ServiceController) StatusText() (string, error) {
	output, err := (command.Runner{}).QuietOutput(
		SystemctlPath,
		"show",
		"--property=StatusText",
		"--value",
		c.unit,
	)
	if err != nil {
		return "", fmt.Errorf("read systemd status for %s: %w", c.unit, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (c ServiceController) queryState(action string, knownStates map[string]bool) (bool, error) {
	output, err := (command.Runner{}).Output(SystemctlPath, action, c.unit)
	state := strings.TrimSpace(string(output))
	// systemctl state queries use this result model:
	//   "active"   + exit 0 -> valid state, true
	//   "inactive" + exit 3 -> valid state, false
	//   "disabled" + exit 1 -> valid state, false
	//   unknown    + error  -> command or systemd failure
	if value, ok := knownStates[state]; ok {
		return value, nil
	}
	if err != nil {
		return false, fmt.Errorf("systemctl %s %s: %w", action, c.unit, err)
	}
	return false, fmt.Errorf("unexpected systemctl %s state %q", action, state)
}

func (c ServiceController) run(action string) error {
	executable := SystemctlPath
	args := []string{action, c.unit}
	if c.privileged {
		executable, args = command.PrivilegedCommand(executable, args...)
	}
	if err := (command.Runner{}).Run(executable, args...); err != nil {
		return fmt.Errorf("systemctl %s %s: %w", action, c.unit, err)
	}
	return nil
}
