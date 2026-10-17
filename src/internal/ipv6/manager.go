package ipv6

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

const (
	ipv6AllKey     = "net.ipv6.conf.all.disable_ipv6"
	ipv6DefaultKey = "net.ipv6.conf.default.disable_ipv6"
	ConfPath       = "/proc/sys/net/ipv6/conf"
	StatePath      = "/var/lib/shadow21/ipv6.state"
	sysctlPath     = "/sbin/sysctl"
)

type ipv6Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ipv6State struct {
	Settings []ipv6Setting `json:"settings"`
}

type Manager struct{}

func (Manager) Capture() error {
	exists, err := fileops.Exists(StatePath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	state, err := currentState()
	if err != nil {
		return err
	}
	content, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return fileops.WriteAtomicWithDir(StatePath, append(content, '\n'), 0600, 0700)
}

func currentState() (ipv6State, error) {
	entries, err := os.ReadDir(ConfPath)
	if err != nil {
		return ipv6State{}, fmt.Errorf("read IPv6 interfaces: %w", err)
	}

	var interfaces []string
	for _, entry := range entries {
		if entry.IsDir() {
			interfaces = append(interfaces, entry.Name())
		}
	}
	sort.SliceStable(interfaces, func(i, j int) bool {
		return ipv6InterfaceOrder(interfaces[i]) < ipv6InterfaceOrder(interfaces[j])
	})

	state := ipv6State{Settings: make([]ipv6Setting, 0, len(interfaces))}
	for _, name := range interfaces {
		path := filepath.Join(ConfPath, name, "disable_ipv6")
		data, err := os.ReadFile(path)
		if err != nil {
			return ipv6State{}, fmt.Errorf("read IPv6 state for %s: %w", name, err)
		}
		value := strings.TrimSpace(string(data))
		if value != "0" && value != "1" {
			return ipv6State{}, fmt.Errorf("read IPv6 state for %s: unexpected value %q", name, value)
		}
		state.Settings = append(state.Settings, ipv6Setting{
			Key:   "net.ipv6.conf." + name + ".disable_ipv6",
			Value: value,
		})
	}
	if err := validateIPv6State(state); err != nil {
		return ipv6State{}, err
	}
	return state, nil
}

func ipv6InterfaceOrder(name string) string {
	switch name {
	case "all":
		return "0"
	case "default":
		return "1"
	default:
		return "2" + name
	}
}

func (m Manager) Disable() error {
	return m.setState(ipv6State{Settings: []ipv6Setting{
		{Key: ipv6AllKey, Value: "1"},
		{Key: ipv6DefaultKey, Value: "1"},
	}})
}

func (m Manager) Restore() error {
	data, err := os.ReadFile(StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read IPv6 state: %w", err)
	}
	state, err := parseIPv6State(string(data))
	if err != nil {
		return err
	}
	if err := m.restoreExistingState(state); err != nil {
		return err
	}
	if err := fileops.RemoveIfExists(StatePath); err != nil {
		return fmt.Errorf("remove IPv6 state: %w", err)
	}
	return nil
}
