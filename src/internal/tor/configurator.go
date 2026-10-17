package tor

import (
	"errors"
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/config"
)

type Configurator struct{}

func (Configurator) Generate() error {
	cfg, err := config.Load(config.SystemPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	options := TorrcOptionsFromConfig(cfg)
	options, err = (BridgeResolver{}).Resolve(options)
	if err != nil {
		return fmt.Errorf("resolve bridges: %w", err)
	}

	rendered, err := RenderTorrc(options)
	if err != nil {
		return fmt.Errorf("render torrc: %w", err)
	}
	if err := (NyxrcManager{}).Ensure(); err != nil {
		return fmt.Errorf("ensure nyxrc: %w", err)
	}

	manager := TorrcManager{}
	if err := manager.Backup(); err != nil && !errors.Is(err, ErrTorrcBackupExists) {
		return fmt.Errorf("backup torrc: %w", err)
	}
	if err := ValidateRendered(rendered); err != nil {
		return err
	}
	if err := manager.Write(rendered); err != nil {
		return fmt.Errorf("write torrc: %w", err)
	}
	return nil
}
