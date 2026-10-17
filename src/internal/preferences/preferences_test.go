package preferences

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAndPersistence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, e := Load()
	if e != nil || s.TmuxEnabled || s.Language != "en" {
		t.Fatalf("bad defaults: %+v %v", s, e)
	}
	s.Language = "de"
	s.TmuxEnabled = true
	if e = Save(s); e != nil {
		t.Fatal(e)
	}
	got, e := Load()
	if e != nil || got != s {
		t.Fatal(got, e)
	}
	p, _ := Path()
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal("unsafe permissions")
	}
}
func TestEndpointValidation(t *testing.T) {
	for _, s := range []string{"0.0.0.0:9050", "192.168.1.2:9050", "localhost:9050", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:9050\nSocksPolicy accept *"} {
		if ValidateEndpoint(s) == nil {
			t.Fatal("accepted", s)
		}
	}
	for _, s := range []string{"127.0.0.1:9050", "[::1]:9050"} {
		if e := ValidateEndpoint(s); e != nil {
			t.Fatal(e)
		}
	}
}
func TestCorruptPreferencesFailClosed(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	os.Mkdir(filepath.Join(base, "shadow21"), 0700)
	os.WriteFile(filepath.Join(base, "shadow21", "preferences.json"), []byte(`{"mode":"wrong"}`), 0600)
	if _, e := Load(); e == nil {
		t.Fatal("invalid settings accepted")
	}
}

func TestAutoIPPreferenceAndLegacySettings(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	p, _ := Path()
	os.MkdirAll(filepath.Dir(p), 0700)
	os.WriteFile(p, []byte(`{"language":"de","mode":"system","endpoint":"127.0.0.1:9050","tmux_enabled":false}`), 0600)
	settings, err := Load()
	if err != nil || !settings.AutoIP {
		t.Fatal(settings, err)
	}
	settings.AutoIP = false
	if err = Save(settings); err != nil {
		t.Fatal(err)
	}
	again, err := Load()
	if err != nil || again.AutoIP {
		t.Fatal(again, err)
	}
}
