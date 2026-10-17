package ipv6

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/command"
	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

func parseIPv6State(content string) (ipv6State, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "{") {
		var state ipv6State
		if err := json.Unmarshal([]byte(content), &state); err != nil {
			return ipv6State{}, fmt.Errorf("invalid IPv6 state: %w", err)
		}
		if err := validateIPv6State(state); err != nil {
			return ipv6State{}, err
		}
		return state, nil
	}

	// State formats:
	//
	// legacy:  "0 1" -> all=0, default=1
	// Recovery can encounter the legacy file after an interrupted upgrade. The
	// fallback accepts exactly two boolean values and restores only what that
	// format recorded instead of inventing per-interface state.
	values := strings.Fields(content)
	if len(values) != 2 {
		return ipv6State{}, fmt.Errorf("invalid IPv6 state")
	}
	for _, value := range values {
		if value != "0" && value != "1" {
			return ipv6State{}, fmt.Errorf("invalid IPv6 state value %q", value)
		}
	}
	return ipv6State{Settings: []ipv6Setting{
		{Key: ipv6AllKey, Value: values[0]},
		{Key: ipv6DefaultKey, Value: values[1]},
	}}, nil
}

func validateIPv6State(state ipv6State) error {
	seen := make(map[string]bool)
	required := map[string]bool{
		ipv6AllKey:     false,
		ipv6DefaultKey: false,
	}
	for _, setting := range state.Settings {
		if _, ok := ipv6InterfaceName(setting.Key); !ok {
			return fmt.Errorf("invalid IPv6 state key %q", setting.Key)
		}
		if setting.Value != "0" && setting.Value != "1" {
			return fmt.Errorf("invalid IPv6 state value %q", setting.Value)
		}
		if seen[setting.Key] {
			return fmt.Errorf("duplicate IPv6 state key %q", setting.Key)
		}
		seen[setting.Key] = true
		if _, ok := required[setting.Key]; ok {
			required[setting.Key] = true
		}
	}
	for key, present := range required {
		if !present {
			return fmt.Errorf("missing IPv6 state key %q", key)
		}
	}
	return nil
}

func (m Manager) restoreExistingState(state ipv6State) error {
	var failures []error
	settings := append([]ipv6Setting{}, state.Settings...)
	sort.SliceStable(settings, func(i, j int) bool {
		return ipv6SettingOrder(settings[i].Key) < ipv6SettingOrder(settings[j].Key)
	})
	for _, setting := range settings {
		if setting.Key != ipv6AllKey && setting.Key != ipv6DefaultKey {
			exists, err := settingExists(setting.Key)
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", setting.Key, err))
				continue
			}
			if !exists {
				continue
			}
		}
		if err := m.writeSetting(setting); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", setting.Key, err))
		}
	}
	return errors.Join(failures...)
}

func ipv6SettingOrder(key string) string {
	switch key {
	case ipv6AllKey:
		return "0"
	case ipv6DefaultKey:
		return "1"
	default:
		return "2" + key
	}
}

func settingExists(key string) (bool, error) {
	name, ok := ipv6InterfaceName(key)
	if !ok {
		return false, fmt.Errorf("invalid IPv6 state key %q", key)
	}
	return fileops.Exists(filepath.Join(ConfPath, name, "disable_ipv6"))
}

func ipv6InterfaceName(key string) (string, bool) {
	const prefix = "net.ipv6.conf."
	const suffix = ".disable_ipv6"
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/= \t\r\n") {
		return "", false
	}
	return name, true
}

func (m Manager) setState(state ipv6State) error {
	var failures []error
	for _, setting := range state.Settings {
		if err := m.writeSetting(setting); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", setting.Key, err))
		}
	}
	return errors.Join(failures...)
}

func (Manager) writeSetting(setting ipv6Setting) error {
	name, ok := ipv6InterfaceName(setting.Key)
	if !ok {
		return fmt.Errorf("invalid IPv6 state key %q", setting.Key)
	}
	key := "net/ipv6/conf/" + name + "/disable_ipv6"
	return (command.Runner{}).Run(sysctlPath, "-w", key+"="+setting.Value)
}
