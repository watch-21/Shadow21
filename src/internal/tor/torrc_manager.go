package tor

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

const (
	TorrcPath              = "/etc/tor/torrc"
	TorrcBackupPath        = "/var/lib/shadow21/torrc.backup"
	TorrcMissingMarkerPath = "/var/lib/shadow21/torrc.missing"
	TorrcHashPath          = "/var/lib/shadow21/torrc.managed-hashes"
)

var (
	ErrEmptyTorrc          = errors.New("torrc content is empty")
	ErrTorrcBackupExists   = errors.New("torrc backup already exists")
	ErrTorrcBackupNotFound = errors.New("torrc backup not found")
	ErrTorrcBackupConflict = errors.New("torrc backup state is inconsistent")
)

type TorrcManager struct{ configPath, backupPath, missingPath, hashPath string }

func (m TorrcManager) paths() (string, string, string, string) {
	a, b, c, d := m.configPath, m.backupPath, m.missingPath, m.hashPath
	if a == "" {
		a = TorrcPath
	}
	if b == "" {
		b = TorrcBackupPath
	}
	if c == "" {
		c = TorrcMissingMarkerPath
	}
	if d == "" {
		d = TorrcHashPath
	}
	return a, b, c, d
}
func configHash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func verifyOwnedConfig(config, backup, hashPath string, missing bool) error {
	info, e := os.Lstat(config)
	if os.IsNotExist(e) && missing {
		return nil
	}
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Tor configuration type changed externally")
	}
	current, e := os.ReadFile(config)
	if e != nil {
		return e
	}
	currentHash := configHash(current)
	if !missing {
		original, e := os.ReadFile(backup)
		if e != nil {
			return e
		}
		if configHash(original) == currentHash {
			return nil
		}
	}
	data, e := os.ReadFile(hashPath)
	if e != nil {
		return fmt.Errorf("Tor ownership journal missing; preserve configuration and recover manually")
	}
	var hashes []string
	if json.Unmarshal(data, &hashes) != nil {
		return fmt.Errorf("invalid Tor ownership journal")
	}
	for _, h := range hashes {
		if h == currentHash {
			return nil
		}
	}
	return fmt.Errorf("Tor configuration changed externally; preserve it and recover manually")
}

func (m TorrcManager) Backup() error {
	configPath, backupPath, missingPath, _ := m.paths()

	backupExists, err := fileops.Exists(backupPath)
	if err != nil {
		return fmt.Errorf("check torrc backup: %w", err)
	}
	markerExists, err := fileops.Exists(missingPath)
	if err != nil {
		return fmt.Errorf("check missing torrc marker: %w", err)
	}
	if backupExists && markerExists {
		return ErrTorrcBackupConflict
	}
	if backupExists || markerExists {
		return ErrTorrcBackupExists
	}

	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Original state      Backup artifact   Restore action
			// torrc exists        torrc.backup      write the saved bytes back
			// torrc is absent     torrc.missing     remove the generated torrc
			// No artifact means no backup transaction exists; both artifacts at
			// once mean corrupted state. The explicit marker is therefore what
			// makes "restore the original absence" possible without guessing.
			if err := os.MkdirAll(filepath.Dir(missingPath), 0700); err != nil {
				return fmt.Errorf("create torrc state directory: %w", err)
			}
			if err := fileops.WriteAtomic(missingPath, []byte("missing\n"), 0600); err != nil {
				return fmt.Errorf("write missing torrc marker: %w", err)
			}
			return nil
		}
		return fmt.Errorf("stat torrc: %w", err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read torrc: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0700); err != nil {
		return fmt.Errorf("create torrc backup directory: %w", err)
	}
	if err := fileops.WriteAtomic(backupPath, content, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write torrc backup: %w", err)
	}
	return nil
}

func (m TorrcManager) Restore() error {
	configPath, backupPath, missingPath, hashPath := m.paths()

	backupExists, err := fileops.Exists(backupPath)
	if err != nil {
		return fmt.Errorf("check torrc backup: %w", err)
	}
	markerExists, err := fileops.Exists(missingPath)
	if err != nil {
		return fmt.Errorf("check missing torrc marker: %w", err)
	}
	if backupExists && markerExists {
		return ErrTorrcBackupConflict
	}

	if backupExists || markerExists {
		if e := verifyOwnedConfig(configPath, backupPath, hashPath, markerExists); e != nil {
			return e
		}
	}
	if backupExists {
		info, err := os.Stat(backupPath)
		if err != nil {
			return fmt.Errorf("stat torrc backup: %w", err)
		}
		content, err := os.ReadFile(backupPath)
		if err != nil {
			return fmt.Errorf("read torrc backup: %w", err)
		}
		if err := fileops.WriteAtomic(configPath, content, info.Mode().Perm()); err != nil {
			return fmt.Errorf("restore torrc backup: %w", err)
		}
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("remove torrc backup: %w", err)
		}
		return fileops.RemoveIfExists(hashPath)
	}
	if !markerExists {
		return ErrTorrcBackupNotFound
	}
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove generated torrc: %w", err)
	}
	if err := os.Remove(missingPath); err != nil {
		return fmt.Errorf("remove missing torrc marker: %w", err)
	}
	return fileops.RemoveIfExists(hashPath)
}

func (m TorrcManager) Write(content string) error {
	configPath, backupPath, missingPath, hashPath := m.paths()

	if strings.TrimSpace(content) == "" {
		return ErrEmptyTorrc
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("create torrc directory: %w", err)
	}
	missing, err := fileops.Exists(missingPath)
	if err != nil {
		return err
	}
	if err = verifyOwnedConfig(configPath, backupPath, hashPath, missing); err != nil {
		return err
	}
	hashes := []string{configHash([]byte(content))}
	if current, e := os.ReadFile(configPath); e == nil {
		hashes = append(hashes, configHash(current))
	}
	journal, _ := json.Marshal(hashes)
	if err = fileops.WriteAtomicWithDir(hashPath, journal, 0600, 0700); err != nil {
		return err
	}
	if err := fileops.WriteAtomic(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write torrc: %w", err)
	}
	return nil
}
