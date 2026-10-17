package dns

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

func (m Manager) RestoreBackupOrDHCP() error {
	state, err := readDNSState()
	if err != nil {
		return err
	}
	if !state.isRestored() {
		restoredBackup, err := m.restoreResolverBackup()
		if err != nil {
			return err
		}
		if !restoredBackup {
			if err := removeManagedResolver(ResolvConfPath); err != nil {
				return fmt.Errorf("remove current DNS settings: %w", err)
			}
		}
		if _, err := m.restoreNetworkManagerHook(); err != nil {
			return fmt.Errorf("restore NetworkManager DNS ownership: %w", err)
		}
		if err := syncDNSDirectories(ResolvConfPath, NMHookScript); err != nil {
			return err
		}
		completed := stateRestored
		if state == stateUntracked {
			completed = stateRestoredStandalone
		}
		if err := writeDNSState(completed); err != nil {
			return err
		}
	}
	return m.cleanupRestore()
}

func (m Manager) finishPendingRestore() error {
	state, err := readDNSState()
	if err != nil {
		return err
	}
	if state.isRestored() {
		return m.cleanupRestore()
	}
	return nil
}

func (Manager) cleanupRestore() error {
	// Persist the completed phase before deleting any recovery data.
	if err := syncDNSDirectories(StatePath); err != nil {
		return err
	}
	backups := []string{ResolvConfBackup, NMHookBackup, NMHookMissingMarker}
	for _, path := range backups {
		if err := fileops.RemoveIfExists(path); err != nil {
			return fmt.Errorf("remove restored DNS backup %s: %w", path, err)
		}
	}
	if err := syncDNSDirectories(backups...); err != nil {
		return err
	}
	// A remaining completed marker only repeats cleanup, never restoration.
	if err := fileops.RemoveIfExists(StatePath); err != nil {
		return fmt.Errorf("remove DNS state: %w", err)
	}
	return nil
}

func syncDNSDirectories(paths ...string) error {
	seen := make(map[string]bool)
	for _, path := range paths {
		directory := filepath.Dir(path)
		if seen[directory] {
			continue
		}
		seen[directory] = true
		if err := fileops.SyncDirectory(directory); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("sync DNS directory %s: %w", directory, err)
		}
	}
	return nil
}
