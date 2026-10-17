package tor

import (
	"os"
	"path/filepath"
	"testing"
)

func configManager(t *testing.T) TorrcManager {
	d := t.TempDir()
	return TorrcManager{filepath.Join(d, "torrc"), filepath.Join(d, "backup"), filepath.Join(d, "missing"), filepath.Join(d, "hashes")}
}
func TestTorrcTransactionRestoresOriginalAndPermissions(t *testing.T) {
	m := configManager(t)
	os.WriteFile(m.configPath, []byte("original\n"), 0640)
	if e := m.Backup(); e != nil {
		t.Fatal(e)
	}
	if e := m.Write("managed\n"); e != nil {
		t.Fatal(e)
	}
	if e := m.Write("restarted\n"); e != nil {
		t.Fatal(e)
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(m.configPath)
	info, _ := os.Stat(m.configPath)
	if string(b) != "original\n" || info.Mode().Perm() != 0640 {
		t.Fatal(string(b), info.Mode())
	}
}
func TestTorrcExternalEditAndSymlinkPreserved(t *testing.T) {
	for _, link := range []bool{false, true} {
		m := configManager(t)
		os.WriteFile(m.configPath, []byte("original\n"), 0600)
		m.Backup()
		m.Write("managed\n")
		if link {
			target := m.configPath + ".external"
			os.WriteFile(target, []byte("external\n"), 0600)
			os.Remove(m.configPath)
			os.Symlink(target, m.configPath)
		} else {
			os.WriteFile(m.configPath, []byte("external\n"), 0600)
		}
		if m.Restore() == nil {
			t.Fatal("external change restored over")
		}
		if m.Write("new\n") == nil {
			t.Fatal("external edit overwritten by restart")
		}
		b, _ := os.ReadFile(m.configPath)
		if string(b) != "external\n" {
			t.Fatal("external change lost")
		}
		if _, e := os.Stat(m.backupPath); e != nil {
			t.Fatal("backup lost", e)
		}
	}
}
func TestTorrcOriginalAbsenceAndInterruptedWrite(t *testing.T) {
	m := configManager(t)
	if e := m.Backup(); e != nil {
		t.Fatal(e)
	}
	if e := m.Write("managed\n"); e != nil {
		t.Fatal(e)
	}
	if e := m.Restore(); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(m.configPath); !os.IsNotExist(e) {
		t.Fatal("absence not restored")
	}
	m = configManager(t)
	os.WriteFile(m.configPath, []byte("original\n"), 0600)
	m.Backup()
	m.Write("managed\n")
	old, _ := os.ReadFile(m.configPath)
	m.Write("new\n")
	os.WriteFile(m.configPath, old, 0600)
	if e := m.Restore(); e != nil {
		t.Fatal("journal did not recover interrupted rename", e)
	}
}
