package installconfig

import (
	"context"
	"errors"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) Manager {
	t.Helper()
	d := t.TempDir()
	m := Manager{ConfigPath: filepath.Join(d, "torrc"), StateDir: filepath.Join(d, "state"), Validate: func(_ context.Context, p string) error {
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		if !strings.Contains(string(b), "SocksPort 9050\nSocksPolicy accept *") {
			t.Fatal(string(b))
		}
		return nil
	}}
	os.WriteFile(m.ConfigPath, []byte("# original\nSafeSocks 1\n"), 0640)
	return m
}
func TestInstallPersistsAndRepeatedInstallKeepsFirstBackup(t *testing.T) {
	m := fixture(t)
	ctx := context.Background()
	before, _ := os.ReadFile(m.ConfigPath)
	if e := m.Apply(ctx); e != nil {
		t.Fatal(e)
	}
	once, _ := os.ReadFile(m.ConfigPath)
	if e := m.Apply(ctx); e != nil {
		t.Fatal(e)
	}
	twice, _ := os.ReadFile(m.ConfigPath)
	backup, _ := os.ReadFile(m.backupPath())
	if string(once) != string(twice) || string(backup) != string(before) {
		t.Fatal("repeated install changed backup or duplicated lines")
	}
	if strings.Count(string(twice), "SocksPort 9050") != 1 || strings.Count(string(twice), "SocksPolicy accept *") != 1 {
		t.Fatal(string(twice))
	}
	info, _ := os.Stat(m.backupPath())
	if info.Mode().Perm() != 0600 {
		t.Fatal("unprotected backup")
	}
	info, _ = os.Stat(m.ConfigPath)
	if info.Mode().Perm() != 0640 {
		t.Fatal("installed permissions changed")
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	restored, _ := os.ReadFile(m.ConfigPath)
	if string(restored) != string(before) {
		t.Fatal("original not restored")
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
}
func TestValidationFailureLeavesOriginalAndNoJournal(t *testing.T) {
	m := fixture(t)
	before, _ := os.ReadFile(m.ConfigPath)
	m.Validate = func(context.Context, string) error { return errors.New("Tor missing or invalid configuration") }
	if m.Apply(context.Background()) == nil {
		t.Fatal("validation failure accepted")
	}
	after, _ := os.ReadFile(m.ConfigPath)
	if string(after) != string(before) {
		t.Fatal("invalid config applied")
	}
	if _, e := os.Stat(m.journalPath()); !os.IsNotExist(e) {
		t.Fatal("journal armed on validation failure")
	}
}
func TestExternalEditsPreservedOnUpgradeAndRemoval(t *testing.T) {
	m := fixture(t)
	if e := m.Apply(context.Background()); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(m.ConfigPath, []byte("# external edit\n"), 0640)
	if m.Apply(context.Background()) == nil || m.Restore() == nil {
		t.Fatal("external edit accepted for overwrite")
	}
	b, _ := os.ReadFile(m.ConfigPath)
	if string(b) != "# external edit\n" {
		t.Fatal("external change lost")
	}
	if _, e := os.Stat(m.backupPath()); e != nil {
		t.Fatal("backup lost")
	}
}
func TestExistingSymlinkAndListenerConflictArePreserved(t *testing.T) {
	for _, value := range []string{"SocksPort 0.0.0.0:9050\n", "%include /etc/tor/custom\n"} {
		m := fixture(t)
		os.WriteFile(m.ConfigPath, []byte(value), 0640)
		if m.Apply(context.Background()) == nil {
			t.Fatal(value)
		}
		b, _ := os.ReadFile(m.ConfigPath)
		if string(b) != value {
			t.Fatal("conflict altered")
		}
	}
	m := fixture(t)
	target := m.ConfigPath + ".original"
	os.Rename(m.ConfigPath, target)
	os.Symlink(target, m.ConfigPath)
	if m.Apply(context.Background()) == nil {
		t.Fatal("symlink overwritten")
	}
}
func TestOriginalAbsenceAndInterruptedInstallation(t *testing.T) {
	m := fixture(t)
	os.Remove(m.ConfigPath)
	if e := m.Apply(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(m.ConfigPath); !os.IsNotExist(e) {
		t.Fatal("absence not restored")
	}
	m = fixture(t)
	before, _ := os.ReadFile(m.ConfigPath)
	m.Apply(context.Background())
	os.WriteFile(m.ConfigPath, before, 0640)
	if e := m.Apply(context.Background()); e != nil {
		t.Fatal("interrupted install not resumed", e)
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
}
func TestLateExternalEditIsNotOverwritten(t *testing.T) {
	for _, reinstall := range []bool{false, true} {
		m := fixture(t)
		if reinstall {
			if e := m.Apply(context.Background()); e != nil {
				t.Fatal(e)
			}
		}
		m.Validate = func(context.Context, string) error {
			return os.WriteFile(m.ConfigPath, []byte("# edited during validation\n"), 0640)
		}
		if m.Apply(context.Background()) == nil {
			t.Fatal("late edit overwritten or false success")
		}
		b, _ := os.ReadFile(m.ConfigPath)
		if string(b) != "# edited during validation\n" {
			t.Fatal("late edit lost")
		}
	}
}
func TestUntrackedBackupAndInterruptedRemoval(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.StateDir, 0700)
	before, _ := os.ReadFile(m.ConfigPath)
	os.WriteFile(m.backupPath(), before, 0600)
	if e := m.Apply(context.Background()); e != nil {
		t.Fatal("safe interrupted backup not resumed", e)
	}
	j, _ := os.ReadFile(m.journalPath())
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	os.WriteFile(m.journalPath(), j, 0600)
	if e := m.Restore(); e != nil {
		t.Fatal("completed restore not resumed", e)
	}
}

func TestProxyStopKeepsInstalledBaseline(t *testing.T) {
	m := fixture(t)
	ctx := context.Background()
	if e := m.Apply(ctx); e != nil {
		t.Fatal(e)
	}
	baseline, _ := os.ReadFile(m.ConfigPath)
	active := map[string]bool{}
	p := proxymode.Manager{StateDir: m.StateDir, RuntimeDir: filepath.Join(m.StateDir, "run"), ConfigPath: m.ConfigPath, PublicPath: filepath.Join(m.StateDir, "public.json"), Preflight: func(context.Context) error { return nil }, Ready: func(context.Context) error { return nil }, BindCheck: func(context.Context) error { return nil }, RoutingCheck: func() error { return nil }}
	p.Execute = func(_ context.Context, cmd string, args ...string) ([]byte, error) {
		if cmd == "systemd-run" {
			for _, a := range args {
				if strings.HasPrefix(a, "--unit=") {
					active[strings.TrimPrefix(a, "--unit=")] = true
				}
			}
		}
		if cmd == "systemctl" && args[0] == "stop" {
			active[args[1]] = false
		}
		if cmd == "systemctl" && args[0] == "show" {
			if active[args[1]] {
				return []byte("active\n"), nil
			}
			return []byte("inactive\n"), nil
		}
		return nil, nil
	}
	if e := p.Start(ctx, 1000, "shadow21-tor"); e != nil {
		t.Fatal(e)
	}
	if e := p.Stop(ctx, 1000, "shadow21-tor"); e != nil {
		t.Fatal(e)
	}
	after, _ := os.ReadFile(m.ConfigPath)
	if string(after) != string(baseline) {
		t.Fatal("Stop discarded installed SOCKS configuration")
	}
	if e := m.Apply(ctx); e != nil {
		t.Fatal("reinstall after Stop failed", e)
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
}
