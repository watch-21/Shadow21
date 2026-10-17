package installconfig

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
	"gitlab.com/parrotsec/packages/anonsurf/internal/proxymode"
)

type Journal struct {
	OriginalExists bool   `json:"original_exists"`
	OriginalMode   uint32 `json:"original_mode"`
	OriginalUID    int    `json:"original_uid"`
	OriginalGID    int    `json:"original_gid"`
	OriginalHash   string `json:"original_hash"`
	InstalledHash  string `json:"installed_hash"`
}
type Manager struct {
	ConfigPath, StateDir string
	Validate             func(context.Context, string) error
}

func New() Manager {
	return Manager{ConfigPath: "/etc/tor/torrc", StateDir: "/var/lib/shadow21", Validate: validate}
}
func (m Manager) journalPath() string { return filepath.Join(m.StateDir, "install-torrc.json") }
func (m Manager) backupPath() string  { return filepath.Join(m.StateDir, "install-torrc.backup") }
func hash(b []byte) string            { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func (m Manager) read() (Journal, error) {
	var j Journal
	b, e := os.ReadFile(m.journalPath())
	if e != nil {
		return j, e
	}
	if json.Unmarshal(b, &j) != nil || j.OriginalMode > 0777 || j.OriginalUID < 0 || j.OriginalGID < 0 || len(j.OriginalHash) != 64 || len(j.InstalledHash) != 64 {
		return j, fmt.Errorf("invalid installation Tor configuration journal")
	}
	return j, nil
}
func (m Manager) current() ([]byte, os.FileInfo, error) {
	info, e := os.Lstat(m.ConfigPath)
	if e != nil {
		return nil, nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("Tor configuration must be a regular file; preserve existing symlinks")
	}
	b, e := os.ReadFile(m.ConfigPath)
	return b, info, e
}
func (m Manager) save(j Journal) error {
	b, e := json.Marshal(j)
	if e != nil {
		return e
	}
	return fileops.WriteAtomicWithDir(m.journalPath(), append(b, '\n'), 0600, 0700)
}
func (m Manager) candidate(ctx context.Context, b []byte) error {
	f, e := os.CreateTemp(m.StateDir, "install-torrc-check-")
	if e != nil {
		return e
	}
	p := f.Name()
	defer os.Remove(p)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return m.Validate(ctx, p)
}
func (m Manager) Apply(ctx context.Context) error {
	if e := os.MkdirAll(m.StateDir, 0700); e != nil {
		return e
	}
	j, e := m.read()
	fresh := os.IsNotExist(e)
	if e != nil && !fresh {
		return e
	}
	current, info, e := m.current()
	missing := os.IsNotExist(e)
	if e != nil && !missing {
		return e
	}
	if !fresh {
		if !missing && hash(current) == j.InstalledHash {
			if e = m.candidate(ctx, current); e != nil {
				return e
			}
			return m.expect(j.InstalledHash, false)
		}
		if (!missing && hash(current) != j.OriginalHash) || (missing && j.OriginalExists) {
			return fmt.Errorf("Tor configuration changed externally; installation leaves it untouched")
		}
		original, e := os.ReadFile(m.backupPath())
		if e != nil {
			return e
		}
		if hash(original) != j.OriginalHash {
			return fmt.Errorf("installation Tor backup checksum mismatch")
		}
		rendered, e := proxymode.RenderConfig(string(original))
		if e != nil {
			return e
		}
		b := []byte(rendered)
		if hash(b) != j.InstalledHash {
			return fmt.Errorf("installation format changed during an interrupted operation; restore first")
		}
		if e = m.candidate(ctx, b); e != nil {
			return e
		}
		if e = m.expect(j.OriginalHash, !j.OriginalExists); e != nil {
			return e
		}
		return m.writeConfig(b, j)
	}
	j = Journal{OriginalExists: !missing, OriginalMode: 0600, OriginalHash: hash(current)}
	if !missing {
		j.OriginalMode = uint32(info.Mode().Perm())
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			j.OriginalUID = int(stat.Uid)
			j.OriginalGID = int(stat.Gid)
		} else {
			return fmt.Errorf("cannot record Tor configuration ownership")
		}
	}
	rendered, e := proxymode.RenderConfig(string(current))
	if e != nil {
		return e
	}
	b := []byte(rendered)
	// Validate before writing either the ownership journal or the live file.
	if e = m.candidate(ctx, b); e != nil {
		return e
	}
	if e = m.expect(j.OriginalHash, !j.OriginalExists); e != nil {
		return e
	}
	if existing, e := os.ReadFile(m.backupPath()); e == nil && hash(existing) != j.OriginalHash {
		return fmt.Errorf("untracked installation Tor backup; recover manually")
	} else if e != nil && !os.IsNotExist(e) {
		return e
	}
	if e = fileops.WriteAtomic(m.backupPath(), current, 0600); e != nil {
		return e
	}
	j.InstalledHash = hash(b)
	if e = m.save(j); e != nil {
		return e
	}
	if e = m.expect(j.OriginalHash, !j.OriginalExists); e != nil {
		return e
	}
	return m.writeConfig(b, j)
}
func (m Manager) Restore() error {
	j, e := m.read()
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	current, _, e := m.current()
	missing := os.IsNotExist(e)
	if e != nil && !missing {
		return e
	}
	if (!missing && hash(current) != j.InstalledHash && hash(current) != j.OriginalHash) || (missing && j.OriginalExists) {
		return fmt.Errorf("Tor configuration changed externally; keep the backup and reconcile it before removal")
	}
	if !(j.OriginalExists && !missing && hash(current) == j.OriginalHash) && !(!j.OriginalExists && missing) {
		b, e := os.ReadFile(m.backupPath())
		if e != nil {
			return e
		}
		if hash(b) != j.OriginalHash {
			return fmt.Errorf("installation Tor backup checksum mismatch")
		}
		if j.OriginalExists {
			if e = m.writeConfig(b, j); e != nil {
				return e
			}
		} else if e = fileops.RemoveIfExists(m.ConfigPath); e != nil {
			return e
		}
	}
	if j.OriginalExists {
		if e = os.Chown(m.ConfigPath, j.OriginalUID, j.OriginalGID); e != nil {
			return e
		}
		if e = os.Chmod(m.ConfigPath, os.FileMode(j.OriginalMode)); e != nil {
			return e
		}
	}
	if e = fileops.RemoveIfExists(m.backupPath()); e != nil {
		return e
	}
	if e = fileops.SyncDirectory(m.StateDir); e != nil {
		return e
	}
	if e = fileops.RemoveIfExists(m.journalPath()); e != nil {
		return e
	}
	return fileops.SyncDirectory(m.StateDir)
}
func (m Manager) expect(expected string, missing bool) error {
	b, _, e := m.current()
	if os.IsNotExist(e) && missing {
		return nil
	}
	if e != nil {
		return e
	}
	if hash(b) != expected {
		return fmt.Errorf("Tor configuration changed externally during validation; preserved")
	}
	return nil
}
func (m Manager) writeConfig(b []byte, j Journal) error {
	dir := filepath.Dir(m.ConfigPath)
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".shadow21-install-")
	if e != nil {
		return e
	}
	p := f.Name()
	defer os.Remove(p)
	fail := func(e error) error { f.Close(); return e }
	if _, e = f.Write(b); e != nil {
		return fail(e)
	}
	if e = f.Chown(j.OriginalUID, j.OriginalGID); e != nil {
		return fail(e)
	}
	if e = f.Chmod(os.FileMode(j.OriginalMode)); e != nil {
		return fail(e)
	}
	if e = f.Sync(); e != nil {
		return fail(e)
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(p, m.ConfigPath); e != nil {
		return e
	}
	return fileops.SyncDirectory(dir)
}

func validate(ctx context.Context, path string) error {
	p, e := exec.LookPath("tor")
	if e != nil {
		return fmt.Errorf("Tor must be installed before configuring SOCKS")
	}
	c := exec.CommandContext(ctx, p, "--verify-config", "-f", path)
	c.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C.UTF-8", "HOME=/root"}
	if e = c.Run(); e != nil {
		return errors.New("Tor configuration validation failed; original file was preserved")
	}
	return nil
}
