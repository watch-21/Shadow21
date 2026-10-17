package preferences

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

type Settings struct {
	Language    string `json:"language"`
	Mode        string `json:"mode"`
	Endpoint    string `json:"endpoint"`
	AutoIP      bool   `json:"auto_ip"`
	TmuxEnabled bool   `json:"tmux_enabled"`
}

func Default() Settings {
	return Settings{Language: "en", Mode: "system", Endpoint: "127.0.0.1:9050", AutoIP: true}
}
func ValidateEndpoint(endpoint string) error {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return errors.New("invalid proxy endpoint")
	}
	ip := net.ParseIP(host)
	n, e := strconv.Atoi(port)
	if ip == nil || !ip.IsLoopback() || e != nil || n < 1 || n > 65535 {
		return errors.New("proxy must use a loopback IP and a valid port")
	}
	return nil
}
func (s Settings) Validate() error {
	if s.Language != "en" && s.Language != "de" {
		return errors.New("language must be en or de")
	}
	if s.Mode != "system" && s.Mode != "proxy" {
		return errors.New("mode must be system or proxy")
	}
	return ValidateEndpoint(s.Endpoint)
}
func Path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "shadow21", "preferences.json"), nil
}
func Load() (Settings, error) {
	s := Default()
	p, err := Path()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err = json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("invalid preferences: %w", err)
	}
	return s, s.Validate()
}
func Save(s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fileops.WriteAtomicWithDir(p, append(b, '\n'), 0600, 0700)
}
