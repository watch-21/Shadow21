package anonsurf

import (
	"errors"
	"fmt"

	"gitlab.com/parrotsec/packages/anonsurf/internal/systemd"
	"gitlab.com/parrotsec/packages/anonsurf/internal/tor"
)

var ErrNotRunning = errors.New("shadow21 is not running")

func (Controller) ChangeIdentity() error {
	active, err := systemd.NewAnonsurfdController().IsActive()
	if err != nil {
		return fmt.Errorf("check shadow21 service: %w", err)
	}
	if !active {
		return ErrNotRunning
	}
	return (tor.IdentityChanger{}).Change()
}
