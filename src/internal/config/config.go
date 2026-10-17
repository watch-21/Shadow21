package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	SystemPath  = "/etc/shadow21/shadow21.cfg"
	sectionName = "Shadow21Config"
)

// ErrSectionNotFound keeps missing config explicit instead of silently running with defaults.
var ErrSectionNotFound = errors.New("Shadow21Config section not found")

type Config struct {
	KillSwitch          bool
	UseSandbox          bool
	BridgeAddress       string
	UseSafeSocks        bool
	AutoBridge          bool
	RejectPlainPorts    bool
	ListPlainPorts      string
	HTTPProxyAddress    string
	HTTPProxyCredential string
	ProxyIsHTTPS        bool
}

// Default mirrors configs/shadow21.cfg so missing optional values keep Parrot's shipped behavior.
func Default() Config {
	return Config{
		KillSwitch:          true,
		UseSandbox:          false,
		BridgeAddress:       "",
		UseSafeSocks:        false,
		AutoBridge:          false,
		RejectPlainPorts:    false,
		ListPlainPorts:      "23,109,110,143",
		HTTPProxyAddress:    "",
		HTTPProxyCredential: "",
		ProxyIsHTTPS:        false,
	}
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Default(), err
	}
	defer file.Close()

	return Parse(file)
}

func Parse(reader io.Reader) (Config, error) {
	cfg := Default()
	scanner := bufio.NewScanner(reader)
	currentSection := ""
	foundSection := false
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := trimComment(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if currentSection == sectionName {
				foundSection = true
			}
			continue
		}

		if currentSection == "" {
			return cfg, fmt.Errorf("line %d: setting outside section", lineNumber)
		}
		// Other sections may be introduced by packaging or local tooling without affecting AnonSurf.
		if currentSection != sectionName {
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return cfg, fmt.Errorf("line %d: expected key=value setting", lineNumber)
		}
		key = strings.TrimSpace(key)
		value, err := parseValue(strings.TrimSpace(rawValue))
		if err != nil {
			return cfg, fmt.Errorf("line %d: parse %s: %w", lineNumber, key, err)
		}
		// Empty values mean "unset" in the shipped config, so they intentionally keep defaults.
		if value == "" {
			continue
		}
		if err := cfg.apply(key, value); err != nil {
			return cfg, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, err
	}
	if !foundSection {
		return cfg, ErrSectionNotFound
	}

	return cfg, nil
}

func (cfg *Config) apply(key, value string) error {
	switch key {
	case "kill_switch":
		return setBool(&cfg.KillSwitch, key, value)
	case "use_sandbox":
		return setBool(&cfg.UseSandbox, key, value)
	case "bridge_address":
		cfg.BridgeAddress = value
	case "use_safe_sock":
		return setBool(&cfg.UseSafeSocks, key, value)
	case "auto_bridge":
		return setBool(&cfg.AutoBridge, key, value)
	case "reject_plain_port":
		return setBool(&cfg.RejectPlainPorts, key, value)
	case "list_plain_ports":
		cfg.ListPlainPorts = value
	case "http_proxy_address":
		cfg.HTTPProxyAddress = value
	case "http_proxy_credential":
		cfg.HTTPProxyCredential = value
	case "proxy_is_https":
		return setBool(&cfg.ProxyIsHTTPS, key, value)
	default:
		// Typos in privacy-related settings should fail loudly instead of weakening the setup.
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

func setBool(target *bool, key, value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid boolean for %s: %q", key, value)
	}
	*target = parsed
	return nil
}

func parseValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, `"`) {
		return strconv.Unquote(value)
	}
	if strings.HasPrefix(value, `'`) {
		if !strings.HasSuffix(value, `'`) || len(value) == 1 {
			return "", errors.New("unterminated quoted value")
		}
		return value[1 : len(value)-1], nil
	}
	return value, nil
}

// trimComment keeps comment markers inside quoted values, such as bridge strings or credentials.
func trimComment(line string) string {
	inQuote := false
	var quote rune

	for index, char := range line {
		if (char == '"' || char == '\'') && (index == 0 || line[index-1] != '\\') {
			if !inQuote {
				inQuote = true
				quote = char
			} else if quote == char {
				inQuote = false
			}
			continue
		}
		if !inQuote && (char == '#' || char == ';') {
			return strings.TrimSpace(line[:index])
		}
	}
	return strings.TrimSpace(line)
}
