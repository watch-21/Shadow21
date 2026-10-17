package dns

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

const (
	ResolvConfBackup = "/etc/shadow21/resolv.conf.bak"
	ResolvConfPath   = "/etc/resolv.conf"
)

var ErrBackupExists = errors.New("resolv.conf backup already exists")

type Manager struct{}

func (m Manager) CreateDHCPOnly() error {
	if err := m.finishPendingRestore(); err != nil {
		return err
	}
	if err := m.backupNetworkManagerHook(); err != nil {
		return fmt.Errorf("backup NetworkManager DNS settings: %w", err)
	}
	if err := fileops.RemoveIfExists(ResolvConfPath); err != nil {
		return fmt.Errorf("remove current DNS settings: %w", err)
	}
	return m.createNetworkManagerHook(NMHookHandleResolvConf)
}

func (m Manager) CreateAddrs(listAddr []string) error {
	if len(listAddr) == 0 {
		return ErrNoValidDNSAddress
	}
	if err := m.finishPendingRestore(); err != nil {
		return err
	}
	if err := m.backupNetworkManagerHook(); err != nil {
		return fmt.Errorf("backup NetworkManager DNS settings: %w", err)
	}
	if err := WriteDNSToSystem(listAddr, ResolvConfPath); err != nil {
		return fmt.Errorf("write DNS settings: %w", err)
	}
	return m.createNetworkManagerHook(NMHookDontManageResolvConf)
}

func (m Manager) restoreResolverBackup() (bool, error) {
	// Relative symlink backups can be dangling in the backup directory.
	info, err := os.Lstat(ResolvConfBackup)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat backup: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if err := fileops.CopySymlinkAtomic(ResolvConfBackup, ResolvConfPath); err != nil {
			return false, fmt.Errorf("restore resolver symlink: %w", err)
		}
		return true, nil
	}
	content, err := os.ReadFile(ResolvConfBackup)
	if err != nil {
		return false, fmt.Errorf("read backup: %w", err)
	}
	if err := fileops.WriteAtomic(ResolvConfPath, content, info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("restore backup: %w", err)
	}
	return true, nil
}

func (m Manager) MakeBackup() error {
	if err := m.finishPendingRestore(); err != nil {
		return err
	}
	if _, err := os.Lstat(ResolvConfBackup); err == nil {
		return ErrBackupExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check resolv.conf backup: %w", err)
	}

	info, err := os.Lstat(ResolvConfPath)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("missing %s", ResolvConfPath)
	}
	if err != nil {
		return fmt.Errorf("stat resolv.conf: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(ResolvConfBackup), 0755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if err := fileops.CopySymlinkAtomic(ResolvConfPath, ResolvConfBackup); err != nil {
			return fmt.Errorf("backup resolver symlink: %w", err)
		}
		return nil
	}
	if err := fileops.CopyAtomic(ResolvConfPath, ResolvConfBackup, info.Mode().Perm()); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	return nil
}

func (m Manager) DHCPAddresses() ([]string, error) {
	addresses, err := NMGetDNSServers()
	if err != nil {
		return nil, err
	}
	return GetListValidAddresses(addresses), nil
}
