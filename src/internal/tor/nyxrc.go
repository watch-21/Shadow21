package tor

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

//go:embed assets/nyxrc
var defaultNyxrc string

const NyxrcPath = "/etc/shadow21/nyxrc"

var ErrEmptyNyxrc = errors.New("nyxrc content is empty")

type NyxrcManager struct{}

func (NyxrcManager) Ensure() error {
	expected := strings.TrimRight(defaultNyxrc, "\n")
	if strings.TrimSpace(expected) == "" {
		return ErrEmptyNyxrc
	}

	current, err := os.ReadFile(NyxrcPath)
	if err == nil && strings.Contains(string(current), expected) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read nyxrc: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(NyxrcPath), 0755); err != nil {
		return fmt.Errorf("create nyxrc directory: %w", err)
	}
	if err := fileops.WriteAtomic(NyxrcPath, []byte(expected+"\n"), 0644); err != nil {
		return fmt.Errorf("write nyxrc: %w", err)
	}
	return nil
}
